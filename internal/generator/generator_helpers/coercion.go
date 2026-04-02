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

// Provides runtime coercion helpers for converting any/interface{} values to specific types.
// These are used when the source type is not known at compile time.

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"piko.sh/piko/wdk/maths"
	"piko.sh/piko/wdk/safeconv"
)

// CoerceToString converts any value to its string representation.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the string representation, or empty string for nil.
func CoerceToString(value any) string {
	if value == nil {
		return ""
	}

	if s, ok := coerceStringDirect(value); ok {
		return s
	}

	if s, ok := coerceNumericToString(value); ok {
		return s
	}

	if s, ok := coerceMathsToString(value); ok {
		return s
	}

	if s, ok := coerceTimeToString(value); ok {
		return s
	}

	if stringer, ok := value.(fmt.Stringer); ok {
		return stringer.String()
	}

	return fmt.Sprintf("%v", value)
}

// CoerceToInt converts any value to an int.
//
// Takes value (any) which is the value to convert.
//
// Returns int which is the converted value, or 0 for invalid conversions.
func CoerceToInt(value any) int {
	return int(CoerceToInt64(value))
}

// CoerceToInt64 converts any value to int64.
//
// Takes value (any) which is the value to convert.
//
// Returns int64 which is the converted value, or 0 for invalid conversions.
func CoerceToInt64(value any) int64 {
	if value == nil {
		return 0
	}

	if i, ok := coerceNumericToInt64(value); ok {
		return i
	}

	if i, ok := coerceSpecialToInt64(value); ok {
		return i
	}

	if i, ok := coerceMathsToInt64(value); ok {
		return i
	}

	return coerceToInt64ViaReflect(value)
}

// CoerceToInt32 converts any value to int32.
//
// Takes value (any) which is the value to convert.
//
// Returns int32 which is the converted value, or 0 for invalid conversions.
func CoerceToInt32(value any) int32 {
	return safeconv.Int64ToInt32(CoerceToInt64(value))
}

// CoerceToInt16 converts any value to int16.
//
// Takes value (any) which is the value to convert.
//
// Returns int16 which is the converted value, or 0 for invalid conversions.
func CoerceToInt16(value any) int16 {
	return safeconv.Int64ToInt16(CoerceToInt64(value))
}

// CoerceToFloat64 converts any value to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or 0.0 for invalid
// conversions.
func CoerceToFloat64(value any) float64 {
	if value == nil {
		return 0.0
	}

	if f, ok := coerceFloatToFloat64(value); ok {
		return f
	}

	if f, ok := coerceIntToFloat64(value); ok {
		return f
	}

	if f, ok := coerceSpecialToFloat64(value); ok {
		return f
	}

	if f, ok := coerceMathsToFloat64(value); ok {
		return f
	}

	return coerceToFloat64ViaReflect(value)
}

// CoerceToFloat32 converts any value to float32.
//
// Takes value (any) which is the value to convert.
//
// Returns float32 which is the converted value, or 0.0 for invalid
// conversions.
func CoerceToFloat32(value any) float32 {
	return float32(CoerceToFloat64(value))
}

// CoerceToBool converts any value to a boolean.
// Uses JavaScript-like truthiness semantics.
//
// Takes value (any) which is the value to convert.
//
// Returns bool which is the truthiness of the input value.
func CoerceToBool(value any) bool {
	if value == nil {
		return false
	}

	if b, ok := coerceBoolDirect(value); ok {
		return b
	}

	if b, ok := coerceNumericToBool(value); ok {
		return b
	}

	if b, ok := coerceMathsToBool(value); ok {
		return b
	}

	if b, ok := coerceTimeToBool(value); ok {
		return b
	}

	return coerceToBoolViaReflect(value)
}

// CoerceToDecimal converts any value to maths.Decimal.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.Decimal which is the converted value, or zero for invalid
// conversions.
func CoerceToDecimal(value any) maths.Decimal {
	if value == nil {
		return maths.ZeroDecimal()
	}

	if d, ok := coerceDecimalDirect(value); ok {
		return d
	}

	if d, ok := coerceIntToDecimal(value); ok {
		return d
	}

	if d, ok := coerceOtherToDecimal(value); ok {
		return d
	}

	return coerceToDecimalViaReflect(value)
}

// CoerceToBigInt converts any value to maths.BigInt.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted value, or zero for invalid
// conversions.
func CoerceToBigInt(value any) maths.BigInt {
	if value == nil {
		return maths.ZeroBigInt()
	}

	if b, ok := coerceBigIntDirect(value); ok {
		return b
	}

	if b, ok := coerceIntToBigInt(value); ok {
		return b
	}

	if b, ok := coerceOtherToBigInt(value); ok {
		return b
	}

	return coerceToBigIntViaReflect(value)
}

// coerceStringDirect handles direct string and bool conversions.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the converted value, or empty if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceStringDirect(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		return strconv.FormatBool(v), true
	default:
		return "", false
	}
}

// coerceNumericToString handles all numeric type conversions to string.
//
// Takes value (any) which is the value to convert to a string.
//
// Returns string which is the decimal string representation of the value.
// Returns bool which indicates whether the conversion was successful.
func coerceNumericToString(value any) (string, bool) {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, baseDecimal), true
	case int32:
		return strconv.FormatInt(int64(v), baseDecimal), true
	case int16:
		return strconv.FormatInt(int64(v), baseDecimal), true
	case int8:
		return strconv.FormatInt(int64(v), baseDecimal), true
	case uint:
		return strconv.FormatUint(uint64(v), baseDecimal), true
	case uint64:
		return strconv.FormatUint(v, baseDecimal), true
	case uint32:
		return strconv.FormatUint(uint64(v), baseDecimal), true
	case uint16:
		return strconv.FormatUint(uint64(v), baseDecimal), true
	case uint8:
		return strconv.FormatUint(uint64(v), baseDecimal), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	default:
		return "", false
	}
}

// coerceMathsToString handles maths.Decimal and maths.BigInt conversions to
// string.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the converted value, or empty if nil or not handled.
// Returns bool which indicates whether the conversion was handled.
func coerceMathsToString(value any) (string, bool) {
	switch v := value.(type) {
	case maths.Decimal:
		return v.MustString(), true
	case *maths.Decimal:
		if v == nil {
			return "", true
		}
		return v.MustString(), true
	case maths.BigInt:
		return v.MustString(), true
	case *maths.BigInt:
		if v == nil {
			return "", true
		}
		return v.MustString(), true
	case maths.Money:
		return v.MustString(), true
	case *maths.Money:
		if v == nil {
			return "", true
		}
		return v.MustString(), true
	default:
		return "", false
	}
}

// coerceTimeToString handles time.Time conversions to string.
//
// Takes value (any) which is the value to convert, expected to be time.Time or
// *time.Time.
//
// Returns string which is the RFC3339 formatted time, or empty if conversion
// fails or the pointer is nil.
// Returns bool which is true if value was a time type, false otherwise.
func coerceTimeToString(value any) (string, bool) {
	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339), true
	case *time.Time:
		if v == nil {
			return "", true
		}
		return v.Format(time.RFC3339), true
	default:
		return "", false
	}
}

// coerceNumericToInt64 handles all numeric type conversions to int64.
//
// Takes value (any) which is the value to convert.
//
// Returns int64 which is the converted value, or zero if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceNumericToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case int16:
		return int64(v), true
	case int8:
		return int64(v), true
	case uint:
		return safeconv.Uint64ToInt64(uint64(v)), true
	case uint64:
		return safeconv.Uint64ToInt64(v), true
	case uint32:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint8:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	default:
		return 0, false
	}
}

// coerceSpecialToInt64 handles bool and string conversions to int64.
//
// Takes value (any) which is the value to convert.
//
// Returns int64 which is the converted value.
// Returns bool which indicates whether the conversion was successful.
func coerceSpecialToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case string:
		i, _ := strconv.ParseInt(v, baseDecimal, 64)
		return i, true
	default:
		return 0, false
	}
}

// coerceMathsToInt64 handles maths.Decimal and maths.BigInt conversions to
// int64.
//
// Takes value (any) which is the value to convert.
//
// Returns int64 which is the converted value, or zero if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceMathsToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case maths.Decimal:
		return v.MustInt64(), true
	case *maths.Decimal:
		if v == nil {
			return 0, true
		}
		return v.MustInt64(), true
	case maths.BigInt:
		return v.MustInt64(), true
	case *maths.BigInt:
		if v == nil {
			return 0, true
		}
		return v.MustInt64(), true
	default:
		return 0, false
	}
}

// coerceToInt64ViaReflect uses reflection to convert a value to int64.
//
// Takes value (any) which is the value to convert.
//
// Returns int64 which is the converted value, or 0 if the type is not numeric.
func coerceToInt64ViaReflect(value any) int64 {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return safeconv.Uint64ToInt64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return int64(rv.Float())
	default:
		return 0
	}
}

// coerceFloatToFloat64 handles float type conversions to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or zero if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceFloatToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	default:
		return 0.0, false
	}
}

// coerceIntToFloat64 handles integer type conversions to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or 0.0 if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceIntToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int16:
		return float64(v), true
	case int8:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint8:
		return float64(v), true
	default:
		return 0.0, false
	}
}

// coerceSpecialToFloat64 handles bool and string conversions to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value (1.0 for true, 0.0 for false,
// parsed value for strings, or 0.0 if conversion fails).
// Returns bool which indicates whether the conversion was handled.
func coerceSpecialToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1.0, true
		}
		return 0.0, true
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f, true
	default:
		return 0.0, false
	}
}

// coerceMathsToFloat64 handles maths.Decimal and maths.BigInt conversions to
// float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or 0.0 if conversion fails or
// the value is nil.
// Returns bool which indicates whether the conversion was successful.
func coerceMathsToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case maths.Decimal:
		return v.MustFloat64(), true
	case *maths.Decimal:
		if v == nil {
			return 0.0, true
		}
		return v.MustFloat64(), true
	case maths.BigInt:
		return float64(v.MustInt64()), true
	case *maths.BigInt:
		if v == nil {
			return 0.0, true
		}
		return float64(v.MustInt64()), true
	default:
		return 0.0, false
	}
}

// coerceToFloat64ViaReflect uses reflection to convert a value to float64.
//
// Takes value (any) which is the value to convert.
//
// Returns float64 which is the converted value, or 0.0 if the type is not
// numeric.
func coerceToFloat64ViaReflect(value any) float64 {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	default:
		return 0.0
	}
}

// coerceBoolDirect handles direct bool and string conversions.
//
// Takes value (any) which is the value to coerce to a boolean.
//
// Returns result (bool) which is the coerced boolean value.
// Returns handled (bool) which indicates whether the conversion was handled.
func coerceBoolDirect(value any) (result bool, handled bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		if v == "" {
			return false, true
		}
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b, true
		}
		return true, true
	default:
		return false, false
	}
}

// coerceNumericToBool handles numeric type conversions to bool.
//
// Takes value (any) which is the value to convert.
//
// Returns result (bool) which is true when the numeric value is non-zero.
// Returns handled (bool) which indicates whether value was a numeric type.
func coerceNumericToBool(value any) (result bool, handled bool) {
	switch v := value.(type) {
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	case int32:
		return v != 0, true
	case int16:
		return v != 0, true
	case int8:
		return v != 0, true
	case uint:
		return v != 0, true
	case uint64:
		return v != 0, true
	case uint32:
		return v != 0, true
	case uint16:
		return v != 0, true
	case uint8:
		return v != 0, true
	case float64:
		return v != 0.0, true
	case float32:
		return v != 0.0, true
	default:
		return false, false
	}
}

// coerceMathsToBool handles maths.Decimal and maths.BigInt conversions to bool.
//
// Takes value (any) which is the value to convert.
//
// Returns result (bool) which is true when the value is non-nil and non-zero.
// Returns handled (bool) which is true when the value was a supported maths type.
func coerceMathsToBool(value any) (result bool, handled bool) {
	switch v := value.(type) {
	case maths.Decimal:
		return !v.MustIsZero(), true
	case *maths.Decimal:
		return v != nil && !v.MustIsZero(), true
	case maths.BigInt:
		return !v.MustIsZero(), true
	case *maths.BigInt:
		return v != nil && !v.MustIsZero(), true
	default:
		return false, false
	}
}

// coerceTimeToBool handles time.Time conversions to bool.
//
// Takes value (any) which is the value to convert.
//
// Returns result (bool) which is true when the time is non-zero.
// Returns handled (bool) which indicates whether the conversion was supported.
func coerceTimeToBool(value any) (result bool, handled bool) {
	switch v := value.(type) {
	case time.Time:
		return !v.IsZero(), true
	case *time.Time:
		return v != nil && !v.IsZero(), true
	default:
		return false, false
	}
}

// coerceToBoolViaReflect uses reflection to convert a value to bool.
//
// Takes value (any) which is the value to convert.
//
// Returns bool which is true for non-zero numbers, non-empty strings, and
// non-nil pointers, interfaces, maps, slices, and channels.
func coerceToBoolViaReflect(value any) bool {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0.0
	case reflect.String:
		return rv.String() != ""
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return !rv.IsNil()
	default:
		return true
	}
}

// coerceDecimalDirect handles direct Decimal and BigInt conversions.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.Decimal which is the converted decimal value.
// Returns bool which indicates whether the conversion was successful.
func coerceDecimalDirect(value any) (maths.Decimal, bool) {
	switch v := value.(type) {
	case maths.Decimal:
		return v, true
	case *maths.Decimal:
		if v == nil {
			return maths.ZeroDecimal(), true
		}
		return *v, true
	case maths.BigInt:
		return v.ToDecimal(), true
	case *maths.BigInt:
		if v == nil {
			return maths.ZeroDecimal(), true
		}
		return v.ToDecimal(), true
	default:
		return maths.Decimal{}, false
	}
}

// coerceIntTo converts an integer value of any type to the target type T.
//
// Takes value (any) which is the value to convert.
// Takes fromInt64 (func(...)) which converts an int64 to the target type.
//
// Returns T which is the converted value, or the zero value if conversion fails.
// Returns bool which indicates whether the conversion was successful.
func coerceIntTo[T any](value any, fromInt64 func(int64) T) (T, bool) {
	switch v := value.(type) {
	case int:
		return fromInt64(int64(v)), true
	case int64:
		return fromInt64(v), true
	case int32:
		return fromInt64(int64(v)), true
	case int16:
		return fromInt64(int64(v)), true
	case int8:
		return fromInt64(int64(v)), true
	case uint:
		return fromInt64(safeconv.Uint64ToInt64(uint64(v))), true
	case uint64:
		return fromInt64(safeconv.Uint64ToInt64(v)), true
	case uint32:
		return fromInt64(int64(v)), true
	case uint16:
		return fromInt64(int64(v)), true
	case uint8:
		return fromInt64(int64(v)), true
	default:
		var zero T
		return zero, false
	}
}

// coerceIntToDecimal converts various Go integer types to maths.Decimal.
//
// Takes value (any) which is the integer value to convert.
//
// Returns maths.Decimal which is the converted decimal value.
// Returns bool which indicates whether the conversion was successful.
func coerceIntToDecimal(value any) (maths.Decimal, bool) {
	return coerceIntTo(value, maths.NewDecimalFromInt)
}

// coerceOtherToDecimal handles float and string conversions to Decimal.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.Decimal which is the converted decimal value.
// Returns bool which indicates whether the conversion succeeded.
func coerceOtherToDecimal(value any) (maths.Decimal, bool) {
	switch v := value.(type) {
	case float64:
		return maths.NewDecimalFromFloat(v), true
	case float32:
		return maths.NewDecimalFromFloat(float64(v)), true
	case string:
		return maths.NewDecimalFromString(v), true
	default:
		return maths.Decimal{}, false
	}
}

// coerceToDecimalViaReflect uses reflection to convert a value to Decimal.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.Decimal which is the converted value, or zero if the type is
// not a supported numeric kind.
func coerceToDecimalViaReflect(value any) maths.Decimal {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return maths.NewDecimalFromInt(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return maths.NewDecimalFromInt(safeconv.Uint64ToInt64(rv.Uint()))
	case reflect.Float32, reflect.Float64:
		return maths.NewDecimalFromFloat(rv.Float())
	default:
		return maths.ZeroDecimal()
	}
}

// coerceBigIntDirect handles direct BigInt and Decimal conversions.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted value.
// Returns bool which indicates whether the conversion succeeded.
func coerceBigIntDirect(value any) (maths.BigInt, bool) {
	switch v := value.(type) {
	case maths.BigInt:
		return v, true
	case *maths.BigInt:
		if v == nil {
			return maths.ZeroBigInt(), true
		}
		return *v, true
	case maths.Decimal:
		return v.ToBigInt(), true
	case *maths.Decimal:
		if v == nil {
			return maths.ZeroBigInt(), true
		}
		return v.ToBigInt(), true
	default:
		return maths.BigInt{}, false
	}
}

// coerceIntToBigInt converts various integer types to maths.BigInt.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted value.
// Returns bool which indicates whether the conversion succeeded.
func coerceIntToBigInt(value any) (maths.BigInt, bool) {
	return coerceIntTo(value, maths.NewBigIntFromInt)
}

// coerceOtherToBigInt handles string conversions to BigInt.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted value.
// Returns bool which indicates whether the conversion succeeded.
func coerceOtherToBigInt(value any) (maths.BigInt, bool) {
	if v, ok := value.(string); ok {
		return maths.NewBigIntFromString(v), true
	}
	return maths.BigInt{}, false
}

// coerceToBigIntViaReflect uses reflection to convert a value to BigInt.
//
// Takes value (any) which is the value to convert.
//
// Returns maths.BigInt which is the converted value, or zero if the type is
// not a supported integer kind.
func coerceToBigIntViaReflect(value any) maths.BigInt {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return maths.NewBigIntFromInt(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return maths.NewBigIntFromInt(safeconv.Uint64ToInt64(rv.Uint()))
	default:
		return maths.ZeroBigInt()
	}
}
