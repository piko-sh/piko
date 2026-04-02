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

package binder

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// dereferencePointer unwraps a pointer value, creating a new value if nil.
//
// Takes currentVal (reflect.Value) which is the value to unwrap.
//
// Returns reflect.Value which is the unwrapped value, or the original if not
// a pointer.
func dereferencePointer(currentVal reflect.Value) reflect.Value {
	if currentVal.Kind() == reflect.Pointer {
		if currentVal.IsNil() {
			currentVal.Set(reflect.New(currentVal.Type().Elem()))
		}
		currentVal = currentVal.Elem()
	}
	return currentVal
}

// dereferenceIndirections dereferences both pointer and interface indirections.
// Use it when working with map[string]any where indexed values are wrapped
// in interface{}.
//
// Takes value (reflect.Value) which is the value to unwrap.
//
// Returns reflect.Value which is the unwrapped concrete value.
func dereferenceIndirections(value reflect.Value) reflect.Value {
	for {
		switch value.Kind() {
		case reflect.Pointer, reflect.Interface:
			if value.IsNil() {
				return value
			}
			value = value.Elem()
		default:
			return value
		}
	}
}

// initialiseNilPointer checks if a value is a nil settable pointer and, if so,
// allocates a new value and dereferences it, bridging the gap between
// dereferenceIndirections (which skips nil pointers) and dereferencePointer
// (which initialises them) when the caller needs to access struct fields
// behind nil pointers.
//
// Takes value (reflect.Value) which is the value to check.
//
// Returns reflect.Value which is the dereferenced value if a nil pointer was
// initialised, or the original value otherwise.
func initialiseNilPointer(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Pointer && value.IsNil() && value.CanSet() {
		value.Set(reflect.New(value.Type().Elem()))
		return value.Elem()
	}
	return value
}

// convertMapKey converts a string to a reflect.Value for use as a map key.
// It handles string, signed integer, and unsigned integer key types.
//
// Takes value (string) which is the string form of the key.
// Takes keyType (reflect.Type) which is the target map key type.
//
// Returns reflect.Value which is the converted key.
// Returns error when parsing fails or the key type is not supported.
func convertMapKey(value string, keyType reflect.Type) (reflect.Value, error) {
	switch keyType.Kind() {
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(i).Convert(keyType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(u).Convert(keyType), nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported map key type: %s", keyType.Kind())
	}
}

// fieldByIndexSafe retrieves a struct field by index path, creating any nil
// pointer fields along the way.
//
// Takes v (reflect.Value) which is the struct value to traverse.
// Takes index ([]int) which is the field path as a slice of indices.
//
// Returns reflect.Value which is the field at the given index path.
func fieldByIndexSafe(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// growSliceToFitIndex makes sure a slice has enough space for a given index.
// It checks the maxSize limit to stop the slice from growing too large.
//
// Takes sliceVal (reflect.Value) which is the slice to grow.
// Takes index (int) which is the index that must be reachable.
// Takes maxSize (int) which is the largest allowed size (0 means no limit).
//
// Returns error when sliceVal is not a slice or index is beyond maxSize.
func growSliceToFitIndex(sliceVal reflect.Value, index int, maxSize int) error {
	if sliceVal.Kind() != reflect.Slice {
		return errors.New("value is not a slice")
	}

	if maxSize > 0 && index >= maxSize {
		return fmt.Errorf("slice index %d exceeds maximum allowed size of %d", index, maxSize)
	}

	if index < sliceVal.Len() {
		return nil
	}

	if index >= sliceVal.Cap() {
		newCap := index + 1
		newSlice := reflect.MakeSlice(sliceVal.Type(), newCap, newCap)
		reflect.Copy(newSlice, sliceVal)
		sliceVal.Set(newSlice)
	}
	sliceVal.SetLen(index + 1)

	return nil
}
