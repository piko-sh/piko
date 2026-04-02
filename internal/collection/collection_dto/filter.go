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

package collection_dto

import (
	"fmt"
	"strings"
	"time"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// FilterOperator defines a comparison type used to match filter values.
type FilterOperator string

// SortOrder defines the direction for sorting results.
type SortOrder string

const (
	// filterFuzzyThreshold is the minimum similarity score for fuzzy matching.
	filterFuzzyThreshold = 0.3

	// FilterOpEquals is the equals operator for exact value matching.
	FilterOpEquals FilterOperator = "eq"

	// FilterOpNotEquals is the not-equals operator for filtering.
	FilterOpNotEquals FilterOperator = "ne"

	// FilterOpGreaterThan checks if a field value is greater than the filter value.
	FilterOpGreaterThan FilterOperator = "gt"

	// FilterOpGreaterEqual matches when a field is greater than or equal to a value.
	FilterOpGreaterEqual FilterOperator = "gte"

	// FilterOpLessThan matches when the field value is less than the filter value.
	FilterOpLessThan FilterOperator = "lt"

	// FilterOpLessEqual matches when a field is less than or equal to the
	// given value.
	FilterOpLessEqual FilterOperator = "lte"

	// FilterOpContains checks if a field value contains the given text.
	FilterOpContains FilterOperator = "contains"

	// FilterOpStartsWith matches when a field value starts with a given prefix.
	FilterOpStartsWith FilterOperator = "startsWith"

	// FilterOpEndsWith is a filter operator that matches when a field ends with
	// the given suffix.
	FilterOpEndsWith FilterOperator = "endsWith"

	// FilterOpIn checks if a field value exists within a given list.
	FilterOpIn FilterOperator = "in"

	// FilterOpNotIn checks if a field value is not in a given list.
	FilterOpNotIn FilterOperator = "notIn"

	// FilterOpExists checks whether a field is present in the metadata.
	FilterOpExists FilterOperator = "exists"

	// FilterOpFuzzyMatch is a filter operator for finding close text matches.
	FilterOpFuzzyMatch FilterOperator = "fuzzy"

	// SortAsc is the sort order for ascending results.
	SortAsc SortOrder = "asc"

	// SortDesc sorts results from highest to lowest value.
	SortDesc SortOrder = "desc"

	// SortRandom is a sort order that arranges items in a random sequence.
	SortRandom SortOrder = "random"
)

// Filter represents a single condition used to query collections.
type Filter struct {
	// Value holds the value to compare against the field.
	Value any

	// Field is the metadata field name to match against.
	Field string

	// Operator specifies how to compare the field and value.
	Operator FilterOperator
}

// FilterGroup combines multiple filters with AND/OR logic.
type FilterGroup struct {
	// Logic controls how filters are combined; must be "AND" or "OR".
	Logic string

	// Filters holds the filter conditions to check.
	Filters []Filter
}

// SortOption specifies how to sort collection items.
type SortOption struct {
	// Field is the name of the metadata field to sort by.
	Field string

	// Order specifies the sort direction: ascending or descending.
	Order SortOrder
}

// PaginationOptions specifies how to split results into pages.
type PaginationOptions struct {
	// Page is the page number, starting from 1.
	Page int

	// PageSize is the number of items per page; 0 disables page-based pagination.
	PageSize int

	// Offset is an alternative to Page for cursor-based pagination.
	Offset int

	// Limit is the maximum number of items to return.
	Limit int
}

// ApplyFilterGroup checks if a content item matches a filter group.
//
// When the group is nil or has no filters, returns true.
//
// Takes item (*ContentItem) which is the content to check against the filters.
// Takes group (*FilterGroup) which defines the filters and their logic.
//
// Returns bool which is true if the item matches the group criteria.
func ApplyFilterGroup(item *ContentItem, group *FilterGroup) bool {
	if group == nil || len(group.Filters) == 0 {
		return true
	}

	if group.Logic == "OR" {
		for _, filter := range group.Filters {
			if applyFilter(item, filter) {
				return true
			}
		}
		return false
	}

	for _, filter := range group.Filters {
		if !applyFilter(item, filter) {
			return false
		}
	}
	return true
}

// applyFilter checks if a content item matches a filter.
//
// Takes item (*ContentItem) which is the content item to test.
// Takes filter (Filter) which specifies the field, operator, and value to match.
//
// Returns bool which is true if the item matches the filter criteria.
func applyFilter(item *ContentItem, filter Filter) bool {
	fieldValue, exists := item.Metadata[filter.Field]

	if filter.Operator == FilterOpExists {
		expectedExists, ok := filter.Value.(bool)
		if !ok {
			return false
		}
		return exists == expectedExists
	}

	if !exists {
		return false
	}

	switch filter.Operator {
	case FilterOpEquals:
		return compareEquals(fieldValue, filter.Value)

	case FilterOpNotEquals:
		return !compareEquals(fieldValue, filter.Value)

	case FilterOpGreaterThan:
		return compareGreaterThan(fieldValue, filter.Value)

	case FilterOpGreaterEqual:
		return compareGreaterThan(fieldValue, filter.Value) || compareEquals(fieldValue, filter.Value)

	case FilterOpLessThan:
		return !compareGreaterThan(fieldValue, filter.Value) && !compareEquals(fieldValue, filter.Value)

	case FilterOpLessEqual:
		return !compareGreaterThan(fieldValue, filter.Value)

	case FilterOpContains:
		return compareContains(fieldValue, filter.Value)

	case FilterOpStartsWith:
		return compareStartsWith(fieldValue, filter.Value)

	case FilterOpEndsWith:
		return compareEndsWith(fieldValue, filter.Value)

	case FilterOpIn:
		return compareIn(fieldValue, filter.Value)

	case FilterOpNotIn:
		return !compareIn(fieldValue, filter.Value)

	case FilterOpFuzzyMatch:
		return compareFuzzy(fieldValue, filter.Value)

	default:
		return false
	}
}

// compareEquals checks whether two values are equal.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns bool which is true if the values are equal, false otherwise.
func compareEquals(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}

	if a == b {
		return true
	}

	aString, aIsString := a.(string)
	bString, bIsString := b.(string)
	if aIsString && bIsString {
		return aString == bString
	}

	aBool, aIsBool := a.(bool)
	bBool, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		return aBool == bBool
	}

	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum != nil && bNum != nil {
		return *aNum == *bNum
	}

	return false
}

// compareGreaterThan checks if a is greater than b.
//
// Supports numeric, string, and time comparisons. For strings, uses
// alphabetical ordering. Returns false if the values cannot be compared.
//
// Takes a (any) which is the left value to compare.
// Takes b (any) which is the right value to compare.
//
// Returns bool which is true if a is greater than b, or false if they
// cannot be compared.
func compareGreaterThan(a, b any) bool {
	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum != nil && bNum != nil {
		return *aNum > *bNum
	}

	aString, aIsString := a.(string)
	bString, bIsString := b.(string)
	if aIsString && bIsString {
		return aString > bString
	}

	aTime := toTime(a)
	bTime := toTime(b)
	if aTime != nil && bTime != nil {
		return aTime.After(*bTime)
	}

	return false
}

// compareContains checks if a string field contains a given substring.
//
// Takes fieldValue (any) which is the field to search within.
// Takes searchValue (any) which is the substring to look for.
//
// Returns bool which is true if fieldValue contains searchValue, or false if
// either value is not a string.
func compareContains(fieldValue, searchValue any) bool {
	fieldString, ok := fieldValue.(string)
	if !ok {
		return false
	}
	searchString, ok := searchValue.(string)
	if !ok {
		return false
	}
	return strings.Contains(fieldString, searchString)
}

// compareStartsWith checks if a string field starts with a given prefix.
//
// Takes fieldValue (any) which is the value to check. It must be a string.
// Takes prefix (any) which is the prefix to match. It must be a string.
//
// Returns bool which is true if fieldValue starts with prefix, or false if
// either value is not a string.
func compareStartsWith(fieldValue, prefix any) bool {
	fieldString, ok := fieldValue.(string)
	if !ok {
		return false
	}
	prefixString, ok := prefix.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(fieldString, prefixString)
}

// compareEndsWith checks if a string field ends with a given suffix.
//
// Takes fieldValue (any) which is the value to check. It must be a string.
// Takes suffix (any) which is the suffix to match. It must be a string.
//
// Returns bool which is true if fieldValue ends with suffix, or false if
// either value is not a string.
func compareEndsWith(fieldValue, suffix any) bool {
	fieldString, ok := fieldValue.(string)
	if !ok {
		return false
	}
	suffixString, ok := suffix.(string)
	if !ok {
		return false
	}
	return strings.HasSuffix(fieldString, suffixString)
}

// compareIn checks if a value exists within an array.
//
// Takes fieldValue (any) which is the value to look for.
// Takes arrayValue (any) which is the array to search.
//
// Returns bool which is true if fieldValue is found in arrayValue.
func compareIn(fieldValue, arrayValue any) bool {
	arr, ok := arrayValue.([]any)
	if !ok {
		strArr, strOk := arrayValue.([]string)
		if !strOk {
			return false
		}
		arr = make([]any, len(strArr))
		for i, v := range strArr {
			arr[i] = v
		}
	}

	for _, item := range arr {
		if compareEquals(fieldValue, item) {
			return true
		}
	}
	return false
}

// toFloat64 converts a value to a float64 pointer.
//
// Takes v (any) which is the value to convert.
//
// Returns *float64 which points to the converted value, or nil if the value
// cannot be converted. Supported types are float64, float32, int, int32, int64,
// and string.
func toFloat64(v any) *float64 {
	switch value := v.(type) {
	case float64:
		return &value
	case float32:
		return new(float64(value))
	case int:
		return new(float64(value))
	case int32:
		return new(float64(value))
	case int64:
		return new(float64(value))
	case string:
		var f float64
		_, err := fmt.Sscanf(value, "%f", &f)
		if err == nil {
			return &f
		}
	}
	return nil
}

// toTime converts a value to a time pointer.
//
// Takes v (any) which is the value to convert. This may be a time.Time or a
// string in RFC3339, date-only (2006-01-02), or datetime (2006-01-02 15:04:05)
// format.
//
// Returns *time.Time which is the converted time, or nil if the conversion
// fails.
func toTime(v any) *time.Time {
	switch value := v.(type) {
	case time.Time:
		return &value
	case string:
		formats := []string{
			time.RFC3339,
			"2006-01-02",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, value); err == nil {
				return &t
			}
		}
	}
	return nil
}

// compareFuzzy checks if two values match using fuzzy text matching.
//
// Takes fieldValue (any) which is the field content to match against.
// Takes searchValue (any) which is the search pattern to find.
//
// Returns bool which is true if the field value matches the search pattern
// within the default threshold (0.3), or false if either value is not a
// string.
func compareFuzzy(fieldValue, searchValue any) bool {
	fieldString, ok := fieldValue.(string)
	if !ok {
		return false
	}

	searchString, ok := searchValue.(string)
	if !ok {
		return false
	}

	matched, _ := linguistics_domain.FuzzyMatch(fieldString, searchString, filterFuzzyThreshold, false)
	return matched
}
