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

package cache_domain

import (
	"piko.sh/piko/internal/cache/cache_dto"
)

// MatchesFilter checks if a value matches a single filter condition using the
// field extractor. Uses zero-alloc direct comparison when possible (avoids
// boxing).
//
// Takes fieldExtractor (*FieldExtractor[V]) which extracts field values.
// Takes value (V) which is the value to check against the filter.
// Takes filter (cache_dto.Filter) which specifies the filter condition to apply.
//
// Returns bool which is true if the value matches the filter condition.
func MatchesFilter[V any](fieldExtractor *FieldExtractor[V], value V, filter cache_dto.Filter) bool {
	if matched, ok := fieldExtractor.CompareFieldDirect(value, filter.Field, filter.Operation, filter.Value, filter.Values); ok {
		return matched
	}

	fieldVal, ok := fieldExtractor.ExtractAny(value, filter.Field)
	if !ok {
		return false
	}

	switch filter.Operation {
	case cache_dto.FilterOpEq:
		return CompareEqual(fieldVal, filter.Value)
	case cache_dto.FilterOpNe:
		return !CompareEqual(fieldVal, filter.Value)
	case cache_dto.FilterOpGt:
		return CompareNumeric(fieldVal, filter.Value) > 0
	case cache_dto.FilterOpGe:
		return CompareNumeric(fieldVal, filter.Value) >= 0
	case cache_dto.FilterOpLt:
		return CompareNumeric(fieldVal, filter.Value) < 0
	case cache_dto.FilterOpLe:
		return CompareNumeric(fieldVal, filter.Value) <= 0
	case cache_dto.FilterOpIn:
		return MatchesIn(fieldVal, filter.Values)
	case cache_dto.FilterOpBetween:
		if len(filter.Values) != 2 {
			return false
		}
		cmpMin := CompareNumeric(fieldVal, filter.Values[0])
		cmpMax := CompareNumeric(fieldVal, filter.Values[1])
		return cmpMin >= 0 && cmpMax <= 0
	case cache_dto.FilterOpPrefix:
		return MatchesPrefix(fieldVal, filter.Value)
	default:
		return false
	}
}

// MatchesAllFilters checks if a value matches all filter conditions.
//
// Takes fieldExtractor (*FieldExtractor[V]) which extracts field values.
// Takes value (V) which is the value to check against the filters.
// Takes filters ([]cache_dto.Filter) which contains the filter conditions.
//
// Returns bool which is true if the value matches all filters.
func MatchesAllFilters[V any](fieldExtractor *FieldExtractor[V], value V, filters []cache_dto.Filter) bool {
	for _, filter := range filters {
		if !MatchesFilter(fieldExtractor, value, filter) {
			return false
		}
	}
	return true
}

// CompareEqual checks if two values are equal.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true if the values are equal by direct comparison or
// by their string form.
func CompareEqual(a, b any) bool {
	if a == b {
		return true
	}
	return ToString(a) == ToString(b)
}

// CompareNumeric compares two values as numbers.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns int which is -1, 0, or 1 showing whether a is less than, equal to,
// or greater than b. Falls back to string comparison if values cannot be
// converted to numbers.
func CompareNumeric(a, b any) int {
	aNum, aOk := ToFloat64(a)
	bNum, bOk := ToFloat64(b)
	if aOk && bOk {
		switch {
		case aNum < bNum:
			return -1
		case aNum > bNum:
			return 1
		default:
			return 0
		}
	}
	aString := ToString(a)
	bString := ToString(b)
	switch {
	case aString < bString:
		return -1
	case aString > bString:
		return 1
	default:
		return 0
	}
}

// MatchesIn checks if fieldVal matches any value in the set.
//
// Takes fieldVal (any) which is the value to search for.
// Takes values ([]any) which is the set of values to match against.
//
// Returns bool which is true if fieldVal equals any value in the set.
func MatchesIn(fieldVal any, values []any) bool {
	for _, v := range values {
		if CompareEqual(fieldVal, v) {
			return true
		}
	}
	return false
}

// MatchesPrefix checks if fieldVal starts with the prefix.
//
// Takes fieldVal (any) which is the value to check.
// Takes prefix (any) which is the prefix to match against.
//
// Returns bool which is true if fieldVal starts with prefix.
func MatchesPrefix(fieldVal, prefix any) bool {
	fieldString := ToString(fieldVal)
	prefixString := ToString(prefix)
	return len(fieldString) >= len(prefixString) && fieldString[:len(prefixString)] == prefixString
}
