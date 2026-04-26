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
	"testing"

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
		{name: "max uint32", input: math.MaxUint32, expected: math.MaxUint32},
		{name: "exceeds max clamps to max", input: math.MaxUint32 + 1, expected: math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToUint32(tt.input)
			if result != tt.expected {
				t.Errorf("IntToUint32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("IntToUint16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("IntToUint8(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
		{name: "exceeds max clamps to max", input: math.MaxInt32 + 1, expected: math.MaxInt32},
		{name: "below min clamps to min", input: math.MinInt32 - 1, expected: math.MinInt32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := safeconv.IntToInt32(tt.input)
			if result != tt.expected {
				t.Errorf("IntToInt32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("IntToInt16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToUint32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToInt32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Uint64ToUint32(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMustIntToUint8(t *testing.T) {
	t.Parallel()

	t.Run("valid values", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.MustIntToUint8(0); result != 0 {
			t.Errorf("MustIntToUint8(0) = %d, want 0", result)
		}
		if result := safeconv.MustIntToUint8(100); result != 100 {
			t.Errorf("MustIntToUint8(100) = %d, want 100", result)
		}
		if result := safeconv.MustIntToUint8(math.MaxUint8); result != math.MaxUint8 {
			t.Errorf("MustIntToUint8(255) = %d, want 255", result)
		}
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
		if result := safeconv.MustIntToUint16(0); result != 0 {
			t.Errorf("MustIntToUint16(0) = %d, want 0", result)
		}
		if result := safeconv.MustIntToUint16(math.MaxUint16); result != math.MaxUint16 {
			t.Errorf("MustIntToUint16(65535) = %d, want 65535", result)
		}
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
		if result := safeconv.MustIntToInt16(0); result != 0 {
			t.Errorf("MustIntToInt16(0) = %d, want 0", result)
		}
		if result := safeconv.MustIntToInt16(math.MaxInt16); result != math.MaxInt16 {
			t.Errorf("MustIntToInt16(MaxInt16) = %d, want %d", result, int16(math.MaxInt16))
		}
		if result := safeconv.MustIntToInt16(math.MinInt16); result != math.MinInt16 {
			t.Errorf("MustIntToInt16(MinInt16) = %d, want %d", result, int16(math.MinInt16))
		}
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
		if result := safeconv.MustUintToUint8(0); result != 0 {
			t.Errorf("MustUintToUint8(0) = %d, want 0", result)
		}
		if result := safeconv.MustUintToUint8(math.MaxUint8); result != math.MaxUint8 {
			t.Errorf("MustUintToUint8(255) = %d, want 255", result)
		}
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
		if result := safeconv.MustUint8ToInt8(0); result != 0 {
			t.Errorf("MustUint8ToInt8(0) = %d, want 0", result)
		}
		if result := safeconv.MustUint8ToInt8(math.MaxInt8); result != math.MaxInt8 {
			t.Errorf("MustUint8ToInt8(127) = %d, want 127", result)
		}
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
		if result := safeconv.MustInt8ToUint8(0); result != 0 {
			t.Errorf("MustInt8ToUint8(0) = %d, want 0", result)
		}
		if result := safeconv.MustInt8ToUint8(math.MaxInt8); result != math.MaxInt8 {
			t.Errorf("MustInt8ToUint8(127) = %d, want 127", result)
		}
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
			if result != tt.expected {
				t.Errorf("Uint64ToInt64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToUint64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToInt(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Uint64ToInt(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("IntToUint64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToInt16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("IntToInt8(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Int64ToUint16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("Uint64ToUint16(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToUint64(t *testing.T) {
	t.Parallel()

	t.Run("int negative clamps to zero", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(-1); result != 0 {
			t.Errorf("ToUint64(int(-1)) = %d, want 0", result)
		}
	})

	t.Run("int positive", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(100); result != 100 {
			t.Errorf("ToUint64(int(100)) = %d, want 100", result)
		}
	})

	t.Run("int8 negative clamps to zero", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(int8(-1)); result != 0 {
			t.Errorf("ToUint64(int8(-1)) = %d, want 0", result)
		}
	})

	t.Run("int8 positive", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(int8(100)); result != 100 {
			t.Errorf("ToUint64(int8(100)) = %d, want 100", result)
		}
	})

	t.Run("uint32 positive", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(uint32(42)); result != 42 {
			t.Errorf("ToUint64(uint32(42)) = %d, want 42", result)
		}
	})

	t.Run("uint64 passthrough", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(uint64(math.MaxUint64)); result != math.MaxUint64 {
			t.Errorf("ToUint64(uint64(MaxUint64)) = %d, want %d", result, uint64(math.MaxUint64))
		}
	})

	t.Run("int zero", func(t *testing.T) {
		t.Parallel()
		if result := safeconv.ToUint64(0); result != 0 {
			t.Errorf("ToUint64(0) = %d, want 0", result)
		}
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
			if result := safeconv.Int32ToInt64(tt.input); result != tt.expected {
				t.Errorf("Int32ToInt64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result := safeconv.Int32ToInt(tt.input); result != tt.expected {
				t.Errorf("Int32ToInt(%d) = %d, want %d", tt.input, result, tt.expected)
			}
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
			if result := safeconv.Uint32ToInt64(tt.input); result != tt.expected {
				t.Errorf("Uint32ToInt64(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}
