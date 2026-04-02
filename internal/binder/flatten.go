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
	"fmt"
	"strconv"
)

// flattenMapToFormData converts a nested map[string]any into a flat
// map[string][]string using bracket notation. This bridges JSON data to the
// existing binder pipeline which expects form data.
//
// Flattening rules:
//   - Top-level keys become simple keys:
//     {"name": "Alice"} -> {"name": ["Alice"]}
//   - Nested maps use bracket notation:
//     {"address": {"city": "London"}} -> {"address['city']": ["London"]}
//   - Arrays use index notation:
//     {"tags": ["a", "b"]} -> {"tags[0]": ["a"], "tags[1]": ["b"]}
//   - Non-string leaf values are converted via fmt.Sprint
//   - nil values are skipped
//
// Takes src (map[string]any) which is the nested map to flatten.
//
// Returns map[string][]string which contains the flattened form data.
func flattenMapToFormData(src map[string]any) map[string][]string {
	result := make(map[string][]string, len(src))
	for key, value := range src {
		flattenValue(result, key, value)
	}
	return result
}

// flattenValue recursively flattens a single value into the result map.
//
// Takes result (map[string][]string) which accumulates the flattened entries.
// Takes prefix (string) which is the current bracket-notation path.
// Takes value (any) which is the value to flatten.
func flattenValue(result map[string][]string, prefix string, value any) {
	if value == nil {
		return
	}

	switch v := value.(type) {
	case map[string]any:
		flattenNestedMap(result, prefix, v)
	case []any:
		flattenSlice(result, prefix, v)
	default:
		result[prefix] = []string{leafToString(v)}
	}
}

// flattenNestedMap flattens a nested map by appending bracket-notation keys.
//
// Takes result (map[string][]string) which accumulates the flattened entries.
// Takes prefix (string) which is the parent path.
// Takes m (map[string]any) which is the nested map to flatten.
func flattenNestedMap(result map[string][]string, prefix string, m map[string]any) {
	for key, value := range m {
		childPrefix := prefix + "['" + key + "']"
		flattenValue(result, childPrefix, value)
	}
}

// flattenSlice flattens a slice by appending index notation.
//
// Takes result (map[string][]string) which accumulates the flattened entries.
// Takes prefix (string) which is the parent path.
// Takes s ([]any) which is the slice to flatten.
func flattenSlice(result map[string][]string, prefix string, s []any) {
	for i, value := range s {
		childPrefix := prefix + "[" + strconv.Itoa(i) + "]"
		flattenValue(result, childPrefix, value)
	}
}

// leafToString converts a leaf value to its string representation.
//
// Takes v (any) which is the value to convert.
//
// Returns string which is the string form of the value.
func leafToString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case bool:
		if value {
			return "true"
		}
		return "false"
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		f64 := float64(value)
		if f64 == float64(int64(f64)) {
			return strconv.FormatInt(int64(f64), 10)
		}
		return strconv.FormatFloat(f64, 'f', -1, 32)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	default:
		return fmt.Sprint(v)
	}
}
