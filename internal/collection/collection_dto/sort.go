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
	"math/rand/v2"
	"slices"
	"strings"
	"time"
)

// PaginationMeta holds details about a paginated set of results.
type PaginationMeta struct {
	// CurrentPage is the current page number, starting from 1.
	CurrentPage int

	// PageSize is the number of items per page.
	PageSize int

	// TotalItems is the total number of items across all pages.
	TotalItems int

	// TotalPages is the total number of pages.
	TotalPages int

	// HasNextPage indicates whether there is a next page of results.
	HasNextPage bool

	// HasPrevPage indicates whether a previous page exists.
	HasPrevPage bool
}

// SortItems sorts a slice of content items based on the given sort options.
//
// This function sorts the slice in place. If you need to keep the original
// order, make a copy of the slice first.
//
// When the first sort option uses SortRandom, the slice is shuffled randomly
// and any later sort options are ignored.
//
// Takes items ([]*ContentItem) which is the slice to sort in place.
// Takes sortOptions ([]SortOption) which specifies the fields and direction
// for sorting.
func SortItems(items []*ContentItem, sortOptions []SortOption) {
	if len(sortOptions) == 0 {
		return
	}

	if sortOptions[0].Order == SortRandom {
		rand.Shuffle(len(items), func(i, j int) {
			items[i], items[j] = items[j], items[i]
		})
		return
	}

	slices.SortFunc(items, func(a, b *ContentItem) int {
		return compareitemsInt(a, b, sortOptions)
	})
}

// PaginateItems applies pagination to a slice of items.
//
// When pagination is nil, returns the original slice unchanged.
//
// Takes items ([]*ContentItem) which is the slice to paginate.
// Takes pagination (*PaginationOptions) which sets page-based or offset-based
// options.
//
// Returns []*ContentItem which is a new slice with the requested page of items.
func PaginateItems(items []*ContentItem, pagination *PaginationOptions) []*ContentItem {
	if pagination == nil {
		return items
	}

	total := len(items)

	var offset, limit int

	if pagination.Page > 0 && pagination.PageSize > 0 {
		offset = (pagination.Page - 1) * pagination.PageSize
		limit = pagination.PageSize
	} else if pagination.Offset >= 0 && pagination.Limit > 0 {
		offset = pagination.Offset
		limit = pagination.Limit
	} else if pagination.Limit > 0 {
		offset = 0
		limit = pagination.Limit
	} else {
		return items
	}

	if offset >= total {
		return []*ContentItem{}
	}

	end := min(offset+limit, total)

	return items[offset:end]
}

// CalculatePaginationMeta calculates pagination metadata.
//
// When pagination is nil or has a zero page size, returns metadata treating all
// items as a single page.
//
// Takes total (int) which is the total number of items to paginate.
// Takes pagination (*PaginationOptions) which specifies the page and page size.
//
// Returns *PaginationMeta which contains the calculated pagination details.
func CalculatePaginationMeta(total int, pagination *PaginationOptions) *PaginationMeta {
	if pagination == nil || pagination.PageSize == 0 {
		return &PaginationMeta{
			CurrentPage: 1,
			PageSize:    total,
			TotalItems:  total,
			TotalPages:  1,
			HasNextPage: false,
			HasPrevPage: false,
		}
	}

	page := max(pagination.Page, 1)

	pageSize := pagination.PageSize
	totalPages := (total + pageSize - 1) / pageSize

	return &PaginationMeta{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  total,
		TotalPages:  totalPages,
		HasNextPage: page < totalPages,
		HasPrevPage: page > 1,
	}
}

// compareitemsInt returns a three-way comparison of two content items using
// the given sort options.
//
// Takes a (*ContentItem) which is the first item to compare.
// Takes b (*ContentItem) which is the second item to compare.
// Takes sortOptions ([]SortOption) which lists the fields and sort order.
//
// Returns int which is negative if a < b, zero if equal, or positive if a > b.
func compareitemsInt(a, b *ContentItem, sortOptions []SortOption) int {
	for _, opt := range sortOptions {
		c := compareField(a, b, opt.Field)

		if c == 0 {
			continue
		}

		if opt.Order == SortDesc {
			return -c
		}
		return c
	}

	return 0
}

// compareField compares a single field between two items.
//
// When a field is missing from an item, that item sorts before items that
// have the field. When both items are missing the field, they are equal.
//
// Takes a (*ContentItem) which is the first item to compare.
// Takes b (*ContentItem) which is the second item to compare.
// Takes field (string) which is the metadata field name to compare.
//
// Returns int which is -1 if a < b, 0 if a == b, or 1 if a > b.
func compareField(a, b *ContentItem, field string) int {
	aVal, aExists := a.Metadata[field]
	bVal, bExists := b.Metadata[field]

	if !aExists && !bExists {
		return 0
	}
	if !aExists {
		return -1
	}
	if !bExists {
		return 1
	}

	return compareValues(aVal, bVal)
}

// compareValues compares two values that may have different types.
//
// It tries to compare values in this order: nil, string, number, time, bool.
// If none of these work, it converts both values to strings and compares them.
//
// Takes a (any) which is the first value to compare.
// Takes b (any) which is the second value to compare.
//
// Returns int which is -1 if a < b, 0 if equal, or 1 if a > b.
func compareValues(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	aString, aIsString := a.(string)
	bString, bIsString := b.(string)
	if aIsString && bIsString {
		return compareStrings(aString, bString)
	}

	aNum := toFloat64(a)
	bNum := toFloat64(b)
	if aNum != nil && bNum != nil {
		return compareNumbers(*aNum, *bNum)
	}

	aTime := toTime(a)
	bTime := toTime(b)
	if aTime != nil && bTime != nil {
		return compareTimes(*aTime, *bTime)
	}

	aBool, aIsBool := a.(bool)
	bBool, bIsBool := b.(bool)
	if aIsBool && bIsBool {
		return compareBools(aBool, bBool)
	}

	return compareStrings(toString(a), toString(b))
}

// compareStrings compares two strings without regard to letter case.
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
//
// Returns int which is -1 if a < b, 1 if a > b, or 0 if equal.
func compareStrings(a, b string) int {
	return compareFoldedStrings(a, b)
}

// compareFoldedStrings performs case-insensitive string comparison without
// allocating lowercased copies. It folds ASCII characters inline and falls
// back to per-rune folding for non-ASCII.
//
// Takes a (string) which is the first string to compare.
// Takes b (string) which is the second string to compare.
//
// Returns int which is -1 if a < b, 0 if equal, or 1 if a > b.
func compareFoldedStrings(a, b string) int {
	minLen := min(len(a), len(b))
	for i := range minLen {
		if result := compareFoldedBytes(a, b, i); result != 0 {
			return result
		}
	}
	return compareLength(len(a), len(b))
}

// compareFoldedBytes compares two bytes at position i after ASCII case folding.
// Falls back to strings.ToLower for non-ASCII bytes.
//
// Takes a (string) which is the first string.
// Takes b (string) which is the second string.
// Takes i (int) which is the byte position to compare.
//
// Returns int which is 0 if the folded bytes are equal, -1 if a[i] < b[i],
// or 1 if a[i] > b[i].
func compareFoldedBytes(a, b string, i int) int {
	ca, cb := foldASCIIByte(a[i]), foldASCIIByte(b[i])
	if ca == cb {
		return 0
	}
	if ca|cb >= 0x80 { //nolint:revive // non-ASCII fallback
		return strings.Compare(strings.ToLower(a[i:]), strings.ToLower(b[i:]))
	}
	if ca < cb {
		return -1
	}
	return 1
}

// foldASCIIByte converts an ASCII uppercase byte to lowercase.
//
// Takes c (byte) which is the byte to fold.
//
// Returns byte which is the lowercase equivalent for ASCII letters, or c
// unchanged for all other bytes.
func foldASCIIByte(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// compareLength returns the three-way comparison of two lengths.
//
// Takes a (int) which is the first length.
// Takes b (int) which is the second length.
//
// Returns int which is -1 if a < b, 1 if a > b, or 0 if equal.
func compareLength(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// compareNumbers compares two numbers and returns their order.
//
// Takes a (float64) which is the first number to compare.
// Takes b (float64) which is the second number to compare.
//
// Returns int which is -1 if a is less than b, 1 if a is greater than b, or 0
// if they are equal.
func compareNumbers(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// compareTimes compares two time values for ordering.
//
// Takes a (time.Time) which is the first time value to compare.
// Takes b (time.Time) which is the second time value to compare.
//
// Returns int which is -1 if a is before b, 1 if a is after b, or 0 if they
// are equal.
func compareTimes(a, b time.Time) int {
	if a.Before(b) {
		return -1
	}
	if a.After(b) {
		return 1
	}
	return 0
}

// compareBools compares two boolean values for sorting.
// False is sorted before true.
//
// Takes a (bool) which is the first boolean to compare.
// Takes b (bool) which is the second boolean to compare.
//
// Returns int which is -1 if a is less than b, 0 if equal, or 1 if a is
// greater than b.
func compareBools(a, b bool) int {
	if a == b {
		return 0
	}
	if !a {
		return -1
	}
	return 1
}

// toString converts a value to its string form.
//
// Takes v (any) which is the value to convert.
//
// Returns string which is the string form of the value. Returns an empty
// string if v is nil or of an unsupported type.
func toString(v any) string {
	if v == nil {
		return ""
	}

	switch value := v.(type) {
	case string:
		return value
	case bool:
		if value {
			return "true"
		}
		return "false"
	case int:
		return string(rune(value)) //nolint:gosec // int-to-rune for codepoint conversion
	default:
		return ""
	}
}
