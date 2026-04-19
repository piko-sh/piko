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

//go:build !safe && !(js && wasm)

package provider_otter

import (
	"reflect"
	"unsafe"

	"piko.sh/piko/internal/cache/cache_dto"
)

// compareFieldDirect performs zero-allocation field comparison without boxing.
// This is the optimised path for filtering; it reads the field via unsafe
// and compares directly, never boxing to interface{}.
//
// Takes value (V) which is the struct value to extract the field from.
// Takes fieldPath (string) which specifies the dot-separated path to
// the field.
// Takes operator (any) which is the comparison operator to apply.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides multiple values for set
// operations.
//
// Returns matched (bool) which is the comparison result.
// Returns ok (bool) which indicates whether the operation succeeded.
func (fe *FieldExtractor[V]) compareFieldDirect(value V, fieldPath string, operator any, targetValue any, targetValues []any) (matched bool, ok bool) {
	t := reflect.TypeFor[V]()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	tKey := typeKey(t)

	fastMap := globalTypeCache.fast.Load()
	if fastMap == nil {
		return false, false
	}

	accessors, ok := (*fastMap)[tKey]
	if !ok {
		return false, false
	}

	accessor, ok := accessors[fieldPath]
	if !ok || !accessor.isDirect {
		return false, false
	}

	var ptr unsafe.Pointer
	var zeroV V
	if reflect.TypeOf(zeroV).Kind() == reflect.Pointer {
		ptr = unsafe.Pointer(*(*unsafe.Pointer)(unsafe.Pointer(&value)))
		if ptr == nil {
			return false, false
		}
	} else {
		ptr = unsafe.Pointer(&value)
	}

	fieldPtr := unsafe.Add(ptr, accessor.offset)

	return fe.compareByKind(fieldPtr, accessor.kind, operator, targetValue, targetValues)
}

// compareByKind compares a field value based on its type without allocation.
//
// Takes fieldPtr (unsafe.Pointer) which points to the field in memory.
// Takes kind (reflect.Kind) which specifies the type of value to compare.
// Takes operator (any) which is the filter operation to apply.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides values for set operations.
//
// Returns matched (bool) which is true when the comparison succeeds.
// Returns ok (bool) which is true when the operation was valid.
func (*FieldExtractor[V]) compareByKind(fieldPtr unsafe.Pointer, kind reflect.Kind, operator any, targetValue any, targetValues []any) (matched bool, ok bool) {
	filterOp, ok := operator.(cache_dto.FilterOp)
	if !ok {
		return false, false
	}

	switch kind {
	case reflect.Float64:
		return compareFloat64Direct(fieldPtr, filterOp, targetValue, targetValues)
	case reflect.Int, reflect.Int64:
		return compareIntDirect(fieldPtr, kind, filterOp, targetValue, targetValues)
	case reflect.String:
		return compareStringDirect(fieldPtr, filterOp, targetValue, targetValues)
	default:
		return false, false
	}
}

// extractWithAccessor extracts a field value using cached unsafe access.
//
// Takes value (V) which is the struct or pointer to struct to extract from.
// Takes accessor (*fieldAccessor) which provides the cached field access path.
//
// Returns any which is the extracted field value, or nil if extraction fails.
// Returns bool which indicates whether the extraction was successful.
//
// Note: intentionally not decomposed into helpers despite high complexity.
// Extracting helper functions causes them to not be inlined, which reintroduces
// allocations in the hot path.
//
//nolint:revive // inlined for zero-alloc
func (*FieldExtractor[V]) extractWithAccessor(value V, accessor *fieldAccessor) (any, bool) {
	if accessor.isDirect {
		var ptr unsafe.Pointer
		var zeroV V
		if reflect.TypeOf(zeroV).Kind() == reflect.Pointer {
			ptr = unsafe.Pointer(*(*unsafe.Pointer)(unsafe.Pointer(&value)))
			if ptr == nil {
				return nil, false
			}
		} else {
			ptr = unsafe.Pointer(&value)
		}

		fieldPtr := unsafe.Add(ptr, accessor.offset)

		switch accessor.kind {
		case reflect.Int:
			return *(*int)(fieldPtr), true
		case reflect.Int8:
			return *(*int8)(fieldPtr), true
		case reflect.Int16:
			return *(*int16)(fieldPtr), true
		case reflect.Int32:
			return *(*int32)(fieldPtr), true
		case reflect.Int64:
			return *(*int64)(fieldPtr), true
		case reflect.Uint:
			return *(*uint)(fieldPtr), true
		case reflect.Uint8:
			return *(*uint8)(fieldPtr), true
		case reflect.Uint16:
			return *(*uint16)(fieldPtr), true
		case reflect.Uint32:
			return *(*uint32)(fieldPtr), true
		case reflect.Uint64:
			return *(*uint64)(fieldPtr), true
		case reflect.Float32:
			return *(*float32)(fieldPtr), true
		case reflect.Float64:
			return *(*float64)(fieldPtr), true
		case reflect.String:
			return *(*string)(fieldPtr), true
		case reflect.Bool:
			return *(*bool)(fieldPtr), true
		default:
			return nil, false
		}
	}

	v := reflect.ValueOf(value)
	v = dereferencePointers(v)

	for _, fieldIndex := range accessor.indexes {
		if v.Kind() != reflect.Struct || fieldIndex >= v.NumField() {
			return nil, false
		}
		v = v.Field(fieldIndex)
		v = dereferencePointers(v)
	}

	if v.CanInterface() {
		return v.Interface(), true
	}
	return nil, false
}

// typeKey extracts a uintptr from a reflect.Type for use as a map key.
// This is safe because type descriptors are fixed and never move or get freed.
//
// Takes t (reflect.Type) which is the type to get a key from.
//
// Returns uintptr which is a unique identifier for the type.
func typeKey(t reflect.Type) uintptr {
	type iface struct {
		_    uintptr
		data uintptr
	}
	return (*iface)(unsafe.Pointer(&t)).data
}

// compareFloat64Direct compares a float64 field without boxing.
//
// Takes fieldPtr (unsafe.Pointer) which points to the float64
// field to compare.
// Takes operator (cache_dto.FilterOp) which specifies the comparison
// operation.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides range bounds for
// between operations.
//
// Returns matched (bool) which is the comparison result.
// Returns ok (bool) which indicates whether the operation
// succeeded.
//
//nolint:revive // switch filter dispatch
func compareFloat64Direct(fieldPtr unsafe.Pointer, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
	fieldVal := *(*float64)(fieldPtr)

	switch operator {
	case cache_dto.FilterOpEq:
		if target, ok := targetValue.(float64); ok {
			return fieldVal == target, true
		}
	case cache_dto.FilterOpNe:
		if target, ok := targetValue.(float64); ok {
			return fieldVal != target, true
		}
	case cache_dto.FilterOpGt:
		if target, ok := targetValue.(float64); ok {
			return fieldVal > target, true
		}
	case cache_dto.FilterOpGe:
		if target, ok := targetValue.(float64); ok {
			return fieldVal >= target, true
		}
	case cache_dto.FilterOpLt:
		if target, ok := targetValue.(float64); ok {
			return fieldVal < target, true
		}
	case cache_dto.FilterOpLe:
		if target, ok := targetValue.(float64); ok {
			return fieldVal <= target, true
		}
	case cache_dto.FilterOpBetween:
		if len(targetValues) != 2 {
			return false, false
		}
		minVal, ok1 := targetValues[0].(float64)
		if !ok1 {
			return false, false
		}
		maxVal, ok2 := targetValues[1].(float64)
		if !ok2 {
			return false, false
		}
		return fieldVal >= minVal && fieldVal <= maxVal, true
	default:
	}
	return false, false
}

// compareIntDirect compares an int or int64 field without boxing.
//
// Takes fieldPtr (unsafe.Pointer) which points to the int or
// int64 field.
// Takes kind (reflect.Kind) which indicates whether the field is
// int or int64.
// Takes operator (cache_dto.FilterOp) which specifies the comparison
// operation.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides range values for
// between operations.
//
// Returns matched (bool) which is the comparison result.
// Returns ok (bool) which indicates whether the operation succeeded.
//
//nolint:revive // switch filter dispatch
func compareIntDirect(fieldPtr unsafe.Pointer, kind reflect.Kind, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
	var fieldVal int64
	if kind == reflect.Int {
		fieldVal = int64(*(*int)(fieldPtr))
	} else {
		fieldVal = *(*int64)(fieldPtr)
	}

	switch operator {
	case cache_dto.FilterOpEq:
		if target, ok := targetValue.(int64); ok {
			return fieldVal == target, true
		}
		if target, ok := targetValue.(int); ok {
			return fieldVal == int64(target), true
		}
	case cache_dto.FilterOpGt:
		if target, ok := targetValue.(int64); ok {
			return fieldVal > target, true
		}
		if target, ok := targetValue.(int); ok {
			return fieldVal > int64(target), true
		}
	case cache_dto.FilterOpGe:
		if target, ok := targetValue.(int64); ok {
			return fieldVal >= target, true
		}
	case cache_dto.FilterOpLt:
		if target, ok := targetValue.(int64); ok {
			return fieldVal < target, true
		}
	case cache_dto.FilterOpLe:
		if target, ok := targetValue.(int64); ok {
			return fieldVal <= target, true
		}
	case cache_dto.FilterOpBetween:
		if len(targetValues) != 2 {
			return false, false
		}
		minVal, ok1 := targetValues[0].(int64)
		if !ok1 {
			return false, false
		}
		maxVal, ok2 := targetValues[1].(int64)
		if !ok2 {
			return false, false
		}
		return fieldVal >= minVal && fieldVal <= maxVal, true
	default:
	}
	return false, false
}

// compareStringDirect compares a string field directly without type wrapping.
//
// Takes fieldPtr (unsafe.Pointer) which points to the string field to compare.
// Takes operator (cache_dto.FilterOp) which specifies the comparison to
// perform.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides values for the In operation.
//
// Returns matched (bool) which is true when the comparison succeeds.
// Returns ok (bool) which is true when the operation was valid.
func compareStringDirect(fieldPtr unsafe.Pointer, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
	fieldVal := *(*string)(fieldPtr)

	switch operator {
	case cache_dto.FilterOpEq:
		if target, ok := targetValue.(string); ok {
			return fieldVal == target, true
		}
	case cache_dto.FilterOpNe:
		if target, ok := targetValue.(string); ok {
			return fieldVal != target, true
		}
	case cache_dto.FilterOpPrefix:
		if target, ok := targetValue.(string); ok {
			return len(fieldVal) >= len(target) && fieldVal[:len(target)] == target, true
		}
	case cache_dto.FilterOpIn:
		for _, v := range targetValues {
			if target, ok := v.(string); ok && fieldVal == target {
				return true, true
			}
		}
		return false, true
	default:
	}
	return false, false
}
