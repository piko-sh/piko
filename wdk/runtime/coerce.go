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

package runtime

import (
	"strings"
	"time"
)

// CoerceInt extracts an integer from a value of unknown type. YAML parsers
// produce int, int64, or float64 depending on the value and library; the
// helper handles all three.
//
// Takes v (any) which is the value to coerce.
//
// Returns int which is the coerced integer value.
// Returns bool which is true when coercion succeeded.
func CoerceInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case int32:
		return int(n), true
	case int16:
		return int(n), true
	case int8:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}

// CoerceInt64 extracts an int64 from a value of unknown type.
//
// Takes v (any) which is the value to coerce.
//
// Returns int64 which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int16:
		return int64(n), true
	case int8:
		return int64(n), true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	default:
		return 0, false
	}
}

// CoerceFloat64 extracts a float64 from a value of unknown type.
//
// Takes v (any) which is the value to coerce.
//
// Returns float64 which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	default:
		return 0, false
	}
}

// CoerceFloat32 extracts a float32 from a value of unknown type.
//
// Takes v (any) which is the value to coerce.
//
// Returns float32 which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceFloat32(v any) (float32, bool) {
	switch n := v.(type) {
	case float32:
		return n, true
	case float64:
		return float32(n), true
	case int:
		return float32(n), true
	case int64:
		return float32(n), true
	case int32:
		return float32(n), true
	default:
		return 0, false
	}
}

// CoerceStringSlice extracts a string slice from a value of unknown type.
// YAML parsers may produce []any containing strings rather than []string
// directly.
//
// Takes v (any) which is the value to coerce.
//
// Returns []string which is the coerced slice.
// Returns bool which is true when coercion succeeded.
func CoerceStringSlice(v any) ([]string, bool) {
	switch s := v.(type) {
	case []string:
		return s, true
	case []any:
		result := make([]string, 0, len(s))
		for _, item := range s {
			str, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, str)
		}
		return result, true
	default:
		return nil, false
	}
}

// CoerceTime extracts a time.Time from a value of unknown type. YAML parsers
// typically produce time.Time directly for date fields, but the value may also
// arrive as a string.
//
// Takes v (any) which is the value to coerce.
//
// Returns time.Time which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed, true
		}
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			return parsed, true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

// signedInt is a type constraint covering all built-in signed integer types.
type signedInt interface {
	~int8 | ~int16 | ~int32 | ~int | ~int64
}

// unsignedInt is a type constraint covering all built-in unsigned integer types.
type unsignedInt interface {
	~uint8 | ~uint16 | ~uint32 | ~uint | ~uint64
}

// CoerceSignedInt extracts a signed integer of any width from a value of
// unknown type. YAML parsers produce int, int64, or float64 depending on the
// value and library; the helper handles all three and converts to the
// target type T.
//
// Takes v (any) which is the value to coerce.
//
// Returns T which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceSignedInt[T signedInt](v any) (T, bool) {
	switch n := v.(type) {
	case int:
		return T(n), true
	case int64:
		return T(n), true
	case int32:
		return T(n), true
	case int16:
		return T(n), true
	case int8:
		return T(n), true
	case float64:
		return T(n), true
	case float32:
		return T(n), true
	default:
		return 0, false
	}
}

// CoerceUnsignedInt extracts an unsigned integer of any width from a value of
// unknown type.
//
// Takes v (any) which is the value to coerce.
//
// Returns T which is the coerced value.
// Returns bool which is true when coercion succeeded.
func CoerceUnsignedInt[T unsignedInt](v any) (T, bool) {
	switch n := v.(type) {
	case int:
		if n < 0 {
			return 0, false
		}
		return T(n), true
	case int64:
		if n < 0 {
			return 0, false
		}
		return T(n), true
	case int32:
		if n < 0 {
			return 0, false
		}
		return T(n), true
	case float64:
		if n < 0 {
			return 0, false
		}
		return T(n), true
	case uint:
		return T(n), true
	case uint64:
		return T(n), true
	case uint32:
		return T(n), true
	case uint16:
		return T(n), true
	case uint8:
		return T(n), true
	default:
		return 0, false
	}
}

// MetadataGet looks up a key in a metadata map, trying an exact match first
// then falling back to a case-insensitive match.
//
// Takes m (map[string]any) which is the metadata map to search.
// Takes key (string) which is the key to look up.
//
// Returns any which is the found value.
// Returns bool which is true when a matching key was found.
func MetadataGet(m map[string]any, key string) (any, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}
