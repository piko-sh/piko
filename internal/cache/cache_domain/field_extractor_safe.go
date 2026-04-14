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

//go:build safe || (js && wasm)

package cache_domain

import (
	"reflect"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/cache/cache_dto"
)

var (
	// typeIDCounter is a monotonic counter for assigning unique IDs to types.
	typeIDCounter atomic.Uint64

	// typeIDMap maps reflect.Type to a stable uintptr identifier.
	typeIDMap sync.Map
)

// extractWithAccessor extracts a field value using cached reflection access.
//
// This is the safe version that always uses reflect.ValueOf and field traversal
// instead of unsafe pointer arithmetic.
//
// Takes value (V) which is the struct or pointer to struct to extract from.
// Takes accessor (*fieldAccessor) which provides the cached field access path.
//
// Returns any which is the extracted field value, or nil if extraction fails.
// Returns bool which indicates whether the extraction was successful.
func (*FieldExtractor[V]) extractWithAccessor(value V, accessor *fieldAccessor) (any, bool) {
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

// compareFieldDirect performs field comparison using reflection.
//
// This is the safe version that extracts the field value via ExtractAny
// (reflection-based) and then compares using type assertions.
//
// Takes value (V) which is the struct value to extract the field from.
// Takes fieldPath (string) which specifies the dot-separated path to the field.
// Takes operator (any) which is the comparison operator to apply.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides multiple values for set operations.
//
// Returns matched (bool) which is the comparison result.
// Returns ok (bool) which indicates whether the operation succeeded.
func (fe *FieldExtractor[V]) compareFieldDirect(value V, fieldPath string, operator any, targetValue any, targetValues []any) (matched bool, ok bool) {
	extracted, extractOK := fe.ExtractAny(value, fieldPath)
	if !extractOK {
		return false, false
	}

	filterOp, ok := operator.(cache_dto.FilterOp)
	if !ok {
		return false, false
	}

	return compareExtractedValue(extracted, filterOp, targetValue, targetValues)
}

// typeKey returns a stable uintptr identifier for a reflect.Type.
//
// This is the safe version that uses a monotonic counter and sync.Map
// instead of extracting the interface pointer via unsafe.
//
// Takes t (reflect.Type) which is the type to get a key from.
//
// Returns uintptr which is a unique identifier for the type.
func typeKey(t reflect.Type) uintptr {
	if id, ok := typeIDMap.Load(t); ok {
		return id.(uintptr)
	}
	id := uintptr(typeIDCounter.Add(1))
	actual, _ := typeIDMap.LoadOrStore(t, id)
	return actual.(uintptr)
}

// compareExtractedValue compares an extracted value against a target using the
// given filter operation. Supports float64, int/int64, and string comparisons
// via type assertions.
//
// Takes extracted (any) which is the value extracted from the struct field.
// Takes operator (cache_dto.FilterOp) which specifies the comparison operation.
// Takes targetValue (any) which is the value to compare against.
// Takes targetValues ([]any) which provides values for set/range operations.
//
// Returns matched (bool) which is the comparison result.
// Returns ok (bool) which indicates whether the operation succeeded.
//
//nolint:revive // type/filter dispatch
func compareExtractedValue(extracted any, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
	switch fieldVal := extracted.(type) {
	case float64:
		return compareFloat64Value(fieldVal, operator, targetValue, targetValues)
	case int:
		return compareIntValue(int64(fieldVal), operator, targetValue, targetValues)
	case int64:
		return compareIntValue(fieldVal, operator, targetValue, targetValues)
	case string:
		return compareStringValue(fieldVal, operator, targetValue, targetValues)
	default:
		return false, false
	}
}

// compareFloat64Value compares a float64 value against a target.
//
// Takes fieldVal (float64) which is the value to compare.
// Takes operator (cache_dto.FilterOp) which specifies the comparison operation.
// Takes targetValue (any) which is the target for single-value operations.
// Takes targetValues ([]any) which provides bounds for range operations.
//
// Returns matched (bool) which indicates whether the comparison succeeded.
// Returns ok (bool) which indicates whether the operation was valid.
//
//nolint:revive // switch dispatch
func compareFloat64Value(fieldVal float64, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
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

// compareIntValue compares an int64 value against a target.
//
// Takes fieldVal (int64) which is the value to compare.
// Takes operator (cache_dto.FilterOp) which specifies the comparison operation.
// Takes targetValue (any) which is the target for single-value comparisons.
// Takes targetValues ([]any) which provides min and max for between operations.
//
// Returns matched (bool) which indicates whether the comparison succeeded.
// Returns ok (bool) which indicates whether the operation was valid.
//
//nolint:revive // cognitive-complexity: switch dispatch
func compareIntValue(fieldVal int64, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
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

// compareStringValue compares a string value against a target using the
// specified filter operation.
//
// Takes fieldVal (string) which is the value to compare.
// Takes operator (cache_dto.FilterOp) which specifies the comparison operation.
// Takes targetValue (any) which is the single target for most operations.
// Takes targetValues ([]any) which is used for the In operation.
//
// Returns matched (bool) which indicates whether the comparison succeeded.
// Returns ok (bool) which indicates whether the operation was valid.
func compareStringValue(fieldVal string, operator cache_dto.FilterOp, targetValue any, targetValues []any) (matched bool, ok bool) {
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
