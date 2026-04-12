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

package interp_domain

import (
	"cmp"
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

// reflectBinaryOp performs a binary operation on two reflect.Values, dispatching on the
// underlying kind to the appropriate operation function.
//
// Takes a (reflect.Value) which specifies the left operand value.
// Takes b (reflect.Value) which specifies the right operand value.
// Takes intOp (func(int64, int64) int64) which provides the operation for integer types,
// or nil to skip.
// Takes floatOp (func(float64, float64) float64) which provides the operation for float
// types, or nil to skip.
// Takes stringOp (func(string, string) string) which provides the operation for string
// types, or nil to skip.
//
// Returns a reflect.Value of the same type as the operands, or an invalid reflect.Value
// if the kind is unsupported.
func reflectBinaryOp(
	a, b reflect.Value,
	intOp func(int64, int64) int64,
	floatOp func(float64, float64) float64,
	stringOp func(string, string) string,
) reflect.Value {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intOp != nil {
			result := intOp(a.Int(), b.Int())
			out := reflect.New(a.Type()).Elem()
			out.SetInt(result)
			return out
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if intOp != nil {
			result := intOp(safeconv.Uint64ToInt64Reinterpret(a.Uint()), safeconv.Uint64ToInt64Reinterpret(b.Uint()))
			out := reflect.New(a.Type()).Elem()
			out.SetUint(safeconv.Int64ToUint64Reinterpret(result))
			return out
		}
	case reflect.Float32, reflect.Float64:
		if floatOp != nil {
			result := floatOp(a.Float(), b.Float())
			out := reflect.New(a.Type()).Elem()
			out.SetFloat(result)
			return out
		}
	case reflect.String:
		if stringOp != nil {
			result := stringOp(a.String(), b.String())
			out := reflect.New(a.Type()).Elem()
			out.SetString(result)
			return out
		}
	default:
	}
	return reflect.Value{}
}

// reflectCompare performs an ordered comparison of two reflect.Values.
//
// Takes a (reflect.Value) which specifies the left value to compare.
// Takes b (reflect.Value) which specifies the right value to compare.
//
// Returns -1 if a < b, 0 if a == b, or 1 if a > b.
func reflectCompare(a, b reflect.Value) int {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp.Compare(a.Int(), b.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return cmp.Compare(a.Uint(), b.Uint())
	case reflect.Float32, reflect.Float64:
		return cmp.Compare(a.Float(), b.Float())
	case reflect.String:
		return cmp.Compare(a.String(), b.String())
	default:
		return 0
	}
}

// boolToInt64 converts a boolean to an int64, returning 1 for true and 0 for false.
//
// Takes b (bool) which specifies the boolean value to convert.
//
// Returns 1 if b is true, or 0 if b is false.
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
