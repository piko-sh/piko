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

package driven_system_symbols

import (
	"fmt"
	"reflect"
)

// Reflective fallbacks for slice-mutating generic stdlib functions (slices.Compact /
// CompactFunc / Delete / DeleteFunc / Grow / Insert / Repeat / Replace). Most modify the
// backing array in place and return a re-sliced view; a few (Concat, Grow with
// insufficient cap) allocate a new slice. Each helper preserves the stdlib's exact
// semantics including trailing zero-fill so callers can't observe stale data through the
// original slice header.

// zeroTrailing fills positions [from, to) of rv with the zero value of the element type.
// Used after in-place compactions and deletions to match slices.Compact's documented
// zero-fill behaviour.
//
// Takes rv (reflect.Value) which must be an addressable slice.
// Takes from (int), to (int) which define the half-open range to zero.
func zeroTrailing(rv reflect.Value, from, to int) {
	zero := reflect.Zero(rv.Type().Elem())
	for i := from; i < to; i++ {
		rv.Index(i).Set(zero)
	}
}

// reflectSlicesCompact removes consecutive duplicate elements in place, returning a
// re-sliced view. Equivalent to slices.Compact.
//
// Takes x (any) which must be a slice of a comparable element type.
//
// Returns the compacted slice.
func reflectSlicesCompact(x any) any {
	rv := requireSlice("slices.Compact", x)
	originalLen := rv.Len()
	if originalLen < 2 {
		return x
	}
	writeIndex := 1
	for readIndex := 1; readIndex < originalLen; readIndex++ {
		current := rv.Index(readIndex).Interface()
		previous := rv.Index(writeIndex - 1).Interface()
		if !reflect.DeepEqual(current, previous) {
			if writeIndex != readIndex {
				rv.Index(writeIndex).Set(rv.Index(readIndex))
			}
			writeIndex++
		}
	}
	zeroTrailing(rv, writeIndex, originalLen)
	return rv.Slice(0, writeIndex).Interface()
}

// reflectSlicesCompactFunc removes consecutive elements equal under eq in place.
// Equivalent to slices.CompactFunc.
//
// Takes x (any) which must be a slice, and eq (func(any, any) bool) which compares two
// element values.
//
// Returns the compacted slice.
func reflectSlicesCompactFunc(x any, eq func(any, any) bool) any {
	rv := requireSlice("slices.CompactFunc", x)
	originalLen := rv.Len()
	if originalLen < 2 {
		return x
	}
	writeIndex := 1
	for readIndex := 1; readIndex < originalLen; readIndex++ {
		current := rv.Index(readIndex).Interface()
		previous := rv.Index(writeIndex - 1).Interface()
		if !eq(current, previous) {
			if writeIndex != readIndex {
				rv.Index(writeIndex).Set(rv.Index(readIndex))
			}
			writeIndex++
		}
	}
	zeroTrailing(rv, writeIndex, originalLen)
	return rv.Slice(0, writeIndex).Interface()
}

// reflectSlicesDelete removes elements [i, j) from x in place, returning a re-sliced
// view. Equivalent to slices.Delete.
//
// Takes s (any) which must be a slice, and i, j (int) which define the deletion range.
//
// Returns the shortened slice.
//
// Panics if s is not a slice or the indices are out of range.
func reflectSlicesDelete(s any, i, j int) any {
	rv := requireSlice("slices.Delete", s)
	length := rv.Len()
	if i < 0 || j < i || j > length {
		panic(fmt.Sprintf("slices.Delete: indices [%d:%d] out of range for length %d", i, j, length))
	}
	if i == j {
		return s
	}
	shift := j - i
	for k := j; k < length; k++ {
		rv.Index(k - shift).Set(rv.Index(k))
	}
	newLen := length - shift
	zeroTrailing(rv, newLen, length)
	return rv.Slice(0, newLen).Interface()
}

// reflectSlicesDeleteFunc removes every element of s for which del returns true, in
// order, in place. Equivalent to slices.DeleteFunc.
//
// Takes s (any) which must be a slice, and del (func(any) bool) which selects elements to
// remove.
//
// Returns the shortened slice.
func reflectSlicesDeleteFunc(s any, del func(any) bool) any {
	rv := requireSlice("slices.DeleteFunc", s)
	originalLen := rv.Len()
	writeIndex := 0
	for readIndex := range originalLen {
		if !del(rv.Index(readIndex).Interface()) {
			if writeIndex != readIndex {
				rv.Index(writeIndex).Set(rv.Index(readIndex))
			}
			writeIndex++
		}
	}
	zeroTrailing(rv, writeIndex, originalLen)
	return rv.Slice(0, writeIndex).Interface()
}

// reflectSlicesGrow ensures s has capacity for at least n additional elements, allocating
// a new backing array when needed. Equivalent to slices.Grow.
//
// Takes s (any) which must be a slice, and n (int) >= 0.
//
// Returns any which is the resulting slice (possibly with a new backing array) at the
// same length.
//
// Panics when n is negative.
func reflectSlicesGrow(s any, n int) any {
	rv := requireSlice("slices.Grow", s)
	if n < 0 {
		panic(fmt.Sprintf("slices.Grow: negative count %d", n))
	}
	if rv.Cap()-rv.Len() < n {
		result := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len()+n)
		reflect.Copy(result, rv)
		return result.Interface()
	}
	return s
}

// reflectSlicesInsert inserts the values in v at position i in s.
//
// The returned slice may share the backing array of s. Equivalent to slices.Insert.
//
// Takes s (any) which must be a slice, i (int) which is the insertion index, and v
// (...any) which are the values to insert (must be assignable to s's element type or
// convertible to it).
//
// Returns the resulting slice.
//
// Panics when i is out of range, or a value in v has an incompatible type.
func reflectSlicesInsert(s any, i int, v ...any) any {
	rv := requireSlice("slices.Insert", s)
	length := rv.Len()
	if i < 0 || i > length {
		panic(fmt.Sprintf("slices.Insert: index %d out of range for length %d", i, length))
	}
	if len(v) == 0 {
		return s
	}
	elemType := rv.Type().Elem()
	insertVals := make([]reflect.Value, len(v))
	for idx, value := range v {
		converted, err := convertToElemType(value, elemType)
		if err != nil {
			panic(fmt.Sprintf("slices.Insert: %s", err.Error()))
		}
		insertVals[idx] = converted
	}
	result := reflect.MakeSlice(rv.Type(), 0, length+len(v))
	result = reflect.AppendSlice(result, rv.Slice(0, i))
	for _, iv := range insertVals {
		result = reflect.Append(result, iv)
	}
	result = reflect.AppendSlice(result, rv.Slice(i, length))
	return result.Interface()
}

// reflectSlicesRepeat returns a new slice that concatenates count copies of x. Equivalent
// to slices.Repeat.
//
// Takes x (any) which must be a slice, and count (int) >= 0.
//
// Returns the repeated slice.
//
// Panics if x is not a slice, count < 0, or the resulting length would overflow.
func reflectSlicesRepeat(x any, count int) any {
	rv := requireSlice("slices.Repeat", x)
	if count < 0 {
		panic(fmt.Sprintf("slices.Repeat: negative count %d", count))
	}
	if count == 0 || rv.Len() == 0 {
		return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
	}
	totalLen := rv.Len() * count
	if totalLen/count != rv.Len() {
		panic("slices.Repeat: length overflow")
	}
	result := reflect.MakeSlice(rv.Type(), 0, totalLen)
	for range count {
		result = reflect.AppendSlice(result, rv)
	}
	return result.Interface()
}

// reflectSlicesReplace replaces s[i:j] with the values in v, returning the resulting
// slice. Equivalent to slices.Replace.
//
// Takes s (any), i (int), j (int), v (...any).
//
// Returns the resulting slice.
//
// Panics if s is not a slice, indices are out of range, or a value in v has an
// incompatible type.
func reflectSlicesReplace(s any, i, j int, v ...any) any {
	rv := requireSlice("slices.Replace", s)
	length := rv.Len()
	if i < 0 || j < i || j > length {
		panic(fmt.Sprintf("slices.Replace: indices [%d:%d] out of range for length %d", i, j, length))
	}
	elemType := rv.Type().Elem()
	insertVals := make([]reflect.Value, len(v))
	for idx, value := range v {
		converted, err := convertToElemType(value, elemType)
		if err != nil {
			panic(fmt.Sprintf("slices.Replace: %s", err.Error()))
		}
		insertVals[idx] = converted
	}
	newLen := length - (j - i) + len(v)
	result := reflect.MakeSlice(rv.Type(), 0, newLen)
	result = reflect.AppendSlice(result, rv.Slice(0, i))
	for _, iv := range insertVals {
		result = reflect.Append(result, iv)
	}
	result = reflect.AppendSlice(result, rv.Slice(j, length))
	return result.Interface()
}

// convertToElemType coerces value to the slice's element type for Insert/Replace
// operations. Strings, integers, floats, and complex numbers convert across width;
// anything else requires assignability.
//
// Takes value (any) which is the candidate, and elemType (reflect.Type) which is the
// target.
//
// Returns the converted reflect.Value or a descriptive error.
func convertToElemType(value any, elemType reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(elemType), nil
	}
	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(elemType) {
		return rv, nil
	}
	if rv.Type().ConvertibleTo(elemType) {
		return rv.Convert(elemType), nil
	}
	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", value, elemType)
}
