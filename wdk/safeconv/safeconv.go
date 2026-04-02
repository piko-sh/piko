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

package safeconv

import (
	"fmt"
	"math"
)

// integer is a type constraint covering all built-in integer types.
type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// IntToUint64 converts an int to a uint64, returning 0 for negative values.
//
// Takes n (int) which is the value to convert.
//
// Returns uint64 which is the converted value, or 0 if n is negative.
func IntToUint64(n int) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

// IntToUint32 converts an int to uint32, keeping the value within bounds.
// Negative values become 0, values above MaxUint32 become MaxUint32.
//
// Takes n (int) which is the value to convert.
//
// Returns uint32 which is the value after bounds checking.
func IntToUint32(n int) uint32 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(n)
}

// IntToUint16 converts an int to uint16, clamping to valid range.
// Negative values return 0, values > MaxUint16 return MaxUint16.
//
// Takes n (int) which is the value to convert.
//
// Returns uint16 which is the clamped value.
func IntToUint16(n int) uint16 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(n)
}

// IntToUint8 converts an int to a uint8, clamping to the valid range.
// Negative values return 0, values above 255 return 255.
//
// Takes n (int) which is the value to convert.
//
// Returns uint8 which is the clamped value.
func IntToUint8(n int) uint8 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(n)
}

// IntToInt32 converts an int to int32, clamping to valid range.
// Values outside [MinInt32, MaxInt32] are clamped to the respective bound.
//
// Takes n (int) which is the value to convert.
//
// Returns int32 which is the clamped value within the valid int32 range.
func IntToInt32(n int) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < math.MinInt32 {
		return math.MinInt32
	}
	return int32(n)
}

// IntToInt16 converts an int to int16, clamping to the valid range.
// Values outside the int16 range are clamped to the nearest bound.
//
// Takes n (int) which is the value to convert.
//
// Returns int16 which is the clamped value.
func IntToInt16(n int) int16 {
	if n > math.MaxInt16 {
		return math.MaxInt16
	}
	if n < math.MinInt16 {
		return math.MinInt16
	}
	return int16(n)
}

// Int64ToInt16 converts an int64 to int16, clamping to the valid range.
// Values outside [MinInt16, MaxInt16] are clamped to the nearest bound.
//
// Takes n (int64) which is the value to convert.
//
// Returns int16 which is the clamped value within the int16 range.
func Int64ToInt16(n int64) int16 {
	if n > math.MaxInt16 {
		return math.MaxInt16
	}
	if n < math.MinInt16 {
		return math.MinInt16
	}
	return int16(n)
}

// IntToInt8 converts an int to int8, clamping values to fit within bounds.
// Values below -128 become -128, and values above 127 become 127.
//
// Takes n (int) which is the value to convert.
//
// Returns int8 which is the clamped value within the range -128 to 127.
func IntToInt8(n int) int8 {
	if n > math.MaxInt8 {
		return math.MaxInt8
	}
	if n < math.MinInt8 {
		return math.MinInt8
	}
	return int8(n)
}

// Int64ToUint32 converts an int64 to uint32, clamping to valid bounds.
// Negative values become 0, and values above MaxUint32 become MaxUint32.
//
// Takes n (int64) which is the value to convert.
//
// Returns uint32 which is the clamped result.
func Int64ToUint32(n int64) uint32 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(n)
}

// Int64ToInt32 converts an int64 to an int32, clamping values to the valid
// range. Values below MinInt32 become MinInt32, and values above MaxInt32
// become MaxInt32.
//
// Takes n (int64) which is the value to convert.
//
// Returns int32 which is the clamped result within the int32 range.
func Int64ToInt32(n int64) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < math.MinInt32 {
		return math.MinInt32
	}
	return int32(n)
}

// Int64ToUint16 converts an int64 to a uint16, clamping to a valid range.
//
// When n is negative, returns 0. When n is greater than 65535, returns 65535.
//
// Takes n (int64) which is the value to convert.
//
// Returns uint16 which is the clamped value.
func Int64ToUint16(n int64) uint16 {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(n)
}

// Uint64ToUint32 converts a uint64 to uint32, clamping to MaxUint32 if the
// value is too large.
//
// Takes n (uint64) which is the value to convert.
//
// Returns uint32 which is the converted value, clamped to math.MaxUint32 if
// the input is larger.
func Uint64ToUint32(n uint64) uint32 {
	if n > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(n)
}

// Uint64ToUint16 converts a uint64 to uint16, limiting the result to MaxUint16
// if the value is too large.
//
// Takes n (uint64) which is the value to convert.
//
// Returns uint16 which is the converted value, limited to MaxUint16 if n is
// greater than the largest uint16.
func Uint64ToUint16(n uint64) uint16 {
	if n > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(n)
}

// Uint64ToUint8 converts a uint64 to uint8, clamping at math.MaxUint8 if the
// value exceeds the maximum uint8 range.
//
// Takes n (uint64) which is the value to convert.
//
// Returns uint8 which is the converted value, limited to MaxUint8 if n is
// greater than the largest uint8.
func Uint64ToUint8(n uint64) uint8 {
	if n > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(n)
}

// Uint64ToInt64 converts a uint64 to int64, capping at math.MaxInt64 if the
// value exceeds the maximum int64 range.
//
// Takes n (uint64) which is the value to convert.
//
// Returns int64 which is the converted value, capped at math.MaxInt64 if n
// is larger than int64 can hold.
func Uint64ToInt64(n uint64) int64 {
	if n > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(n)
}

// Int64ToUint64 converts an int64 to a uint64.
//
// When n is negative, returns 0.
//
// Takes n (int64) which is the value to convert.
//
// Returns uint64 which is the converted value, or 0 if n is negative.
func Int64ToUint64(n int64) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

// Int64ToInt converts an int64 to an int, clamping to the valid int range.
// On 64-bit systems this is a simple conversion; on 32-bit systems values are
// clamped to fit.
//
// Takes n (int64) which is the value to convert.
//
// Returns int which is the converted value, clamped if needed.
func Int64ToInt(n int64) int {
	if n > math.MaxInt {
		return math.MaxInt
	}
	if n < math.MinInt {
		return math.MinInt
	}
	return int(n)
}

// Uint64ToInt converts a uint64 to an int, clamping to MaxInt if the value is
// too large.
//
// Takes n (uint64) which is the value to convert.
//
// Returns int which is the converted value, clamped to math.MaxInt if n is
// greater than the maximum int value.
func Uint64ToInt(n uint64) int {
	if n > math.MaxInt {
		return math.MaxInt
	}
	return int(n)
}

// IntToUintptr converts an int to uintptr, returning 0 for negative values.
//
// Takes n (int) which is the value to convert.
//
// Returns uintptr which is the converted value, or 0 if n is negative.
func IntToUintptr(n int) uintptr {
	if n < 0 {
		return 0
	}
	return uintptr(n)
}

// MustIntToUint8 converts an int to uint8, panicking if the value is
// outside [0, 255]. Use this when the value must fit and overflow
// would be a programming error.
//
// Takes n (int) which is the value to convert.
//
// Returns uint8 which is the converted value.
//
// Panics if n is negative or greater than 255.
func MustIntToUint8(n int) uint8 {
	if n < 0 || n > math.MaxUint8 {
		panic(fmt.Sprintf("safeconv: int value %d overflows uint8", n))
	}
	return uint8(n) //nolint:gosec // bounds checked above
}

// MustIntToUint16 converts an int to uint16, panicking if the value
// is outside [0, 65535]. Use this when the value must fit and
// overflow would be a programming error.
//
// Takes n (int) which is the value to convert.
//
// Returns uint16 which is the converted value.
//
// Panics if n is negative or greater than 65535.
func MustIntToUint16(n int) uint16 {
	if n < 0 || n > math.MaxUint16 {
		panic(fmt.Sprintf("safeconv: int value %d overflows uint16", n))
	}
	return uint16(n) //nolint:gosec // bounds checked above
}

// MustIntToInt16 converts an int to int16, panicking if the value is
// outside [-32768, 32767]. Use this when the value must fit and
// overflow would be a programming error.
//
// Takes n (int) which is the value to convert.
//
// Returns int16 which is the converted value.
//
// Panics if n is less than -32768 or greater than 32767.
func MustIntToInt16(n int) int16 {
	if n < math.MinInt16 || n > math.MaxInt16 {
		panic(fmt.Sprintf("safeconv: int value %d overflows int16", n))
	}
	return int16(n) //nolint:gosec // bounds checked above
}

// MustUintToUint8 converts a uint to uint8, panicking if the value
// exceeds 255. Use this when the value must fit and overflow would
// be a programming error.
//
// Takes n (uint) which is the value to convert.
//
// Returns uint8 which is the converted value.
//
// Panics if n is greater than 255.
func MustUintToUint8(n uint) uint8 {
	if n > math.MaxUint8 {
		panic(fmt.Sprintf("safeconv: uint value %d overflows uint8", n))
	}
	return uint8(n) //nolint:gosec // bounds checked above
}

// MustUint8ToInt8 converts a uint8 to int8, panicking if the value
// exceeds 127. Use this when the value must fit and overflow would
// be a programming error.
//
// Takes n (uint8) which is the value to convert.
//
// Returns int8 which is the converted value.
//
// Panics if n is greater than 127.
func MustUint8ToInt8(n uint8) int8 {
	if n > math.MaxInt8 {
		panic(fmt.Sprintf("safeconv: uint8 value %d overflows int8", n))
	}
	return int8(n) //nolint:gosec // bounds checked above
}

// MustInt8ToUint8 converts an int8 to uint8, panicking if the value
// is negative. Use this when the value must be non-negative and a
// negative value would be a programming error.
//
// Takes n (int8) which is the value to convert.
//
// Returns uint8 which is the converted value.
//
// Panics if n is negative.
func MustInt8ToUint8(n int8) uint8 {
	if n < 0 {
		panic(fmt.Sprintf("safeconv: int8 value %d overflows uint8", n))
	}
	return uint8(n) //nolint:gosec // bounds checked above
}

// ToUint64 converts any integer type to uint64, clamping negative values to 0.
//
// Use it in cross-platform code where a value may be signed on one platform
// and unsigned on another. For example, syscall.Statfs_t.Bsize is
// int64 on Linux but uint32 on macOS.
//
// Takes n (T) which is the integer value to convert.
//
// Returns uint64 which is the converted value, or 0 if n is negative.
func ToUint64[T integer](n T) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

// Uint16ToInt16 reinterprets a uint16 as int16 via two's complement. Use this
// for binary formats (TrueType, OpenType) that store signed values in
// unsigned fields.
//
// Takes n (uint16) which is the unsigned value to reinterpret.
//
// Returns int16 which is the two's complement interpretation.
func Uint16ToInt16(n uint16) int16 {
	return int16(n) //nolint:gosec // intentional reinterpretation
}

// Int16ToUint16 reinterprets an int16 as uint16 via two's complement. Use this
// for serialising signed TrueType fields into unsigned binary encodings.
//
// Takes n (int16) which is the signed value to reinterpret.
//
// Returns uint16 which is the two's complement encoding.
func Int16ToUint16(n int16) uint16 {
	return uint16(n) //nolint:gosec // intentional reinterpretation
}

// Int32ToUint32 converts an int32 to uint32, returning 0 for negative values.
//
// Takes n (int32) which is the value to convert.
//
// Returns uint32 which is the converted value, or 0 if n is negative.
func Int32ToUint32(n int32) uint32 {
	if n < 0 {
		return 0
	}
	return uint32(n) //nolint:gosec // bounds checked above
}

// Int16ToByte converts an int16 to a byte, clamping to [0, 255].
// Values below 0 become 0, values above 255 become 255.
//
// Takes n (int16) which is the value to convert.
//
// Returns byte which is the clamped value.
func Int16ToByte(n int16) byte {
	if n < 0 {
		return 0
	}
	if n > math.MaxUint8 {
		return math.MaxUint8
	}
	return byte(n) //nolint:gosec // bounds checked above
}

// Uint32ToByte converts a uint32 to byte, clamping to [0, 255].
//
// Takes n (uint32) which is the value to convert.
//
// Returns byte which is the clamped value.
func Uint32ToByte(n uint32) byte {
	if n > math.MaxUint8 {
		return math.MaxUint8
	}
	return byte(n) //nolint:gosec // bounds checked above
}

// RuneToUint16 converts a rune to uint16, clamping to [0, 65535].
// Negative runes become 0, runes above MaxUint16 become MaxUint16.
//
// Takes r (rune) which is the value to convert.
//
// Returns uint16 which is the clamped value.
func RuneToUint16(r rune) uint16 {
	if r < 0 {
		return 0
	}
	if r > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(r) //nolint:gosec // bounds checked above
}

// RuneToByte converts a rune to byte, clamping to [0, 255].
//
// Takes r (rune) which is the value to convert.
//
// Returns byte which is the clamped value.
func RuneToByte(r rune) byte {
	if r < 0 {
		return 0
	}
	if r > math.MaxUint8 {
		return math.MaxUint8
	}
	return byte(r) //nolint:gosec // bounds checked above
}

// Uint32ToInt16 extracts the integer portion of a value stored as uint32,
// clamping to [MinInt16, MaxInt16]. Useful for fixed-point values where
// the uint32 holds a 16.16 representation.
//
// Takes n (uint32) which is the value to convert.
//
// Returns int16 which is the clamped value.
func Uint32ToInt16(n uint32) int16 {
	v := int64(n)
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(v) //nolint:gosec // bounds checked above
}
