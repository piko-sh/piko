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

package collection_domain

import (
	"piko.sh/piko/internal/collection/collection_dto"
)

// applyQueryOptions applies filtering, sorting, and pagination to items.
//
// Takes items ([]collection_dto.ContentItem) which are the items to process.
// Takes options (*collection_dto.FetchOptions) which specifies the query
// options including locale, filters, sorting, and pagination settings.
//
// Returns []collection_dto.ContentItem which contains the filtered, sorted,
// and paginated items.
func (*collectionService) applyQueryOptions(
	items []collection_dto.ContentItem,
	options *collection_dto.FetchOptions,
) []collection_dto.ContentItem {
	itemPtrs := convertToPointerSlice(items)
	itemPtrs = applyLocaleFilter(itemPtrs, options.Locale)
	itemPtrs = applyCustomFilters(itemPtrs, options.Filters)
	applySorting(itemPtrs, options.Filters)
	itemPtrs = applyPagination(itemPtrs, options.Filters)
	return convertToValueSlice(itemPtrs)
}

// convertToPointerSlice changes a slice of values into a slice of pointers.
//
// Takes items ([]collection_dto.ContentItem) which is the slice of values to
// change.
//
// Returns []*collection_dto.ContentItem which holds pointers to each element
// in the original slice.
func convertToPointerSlice(items []collection_dto.ContentItem) []*collection_dto.ContentItem {
	itemPtrs := make([]*collection_dto.ContentItem, len(items))
	for i := range items {
		itemPtrs[i] = &items[i]
	}
	return itemPtrs
}

// convertToValueSlice converts a slice of pointers to a slice of values.
//
// Takes itemPtrs ([]*collection_dto.ContentItem) which is the pointer slice to
// convert.
//
// Returns []collection_dto.ContentItem which contains a copy of each item.
func convertToValueSlice(itemPtrs []*collection_dto.ContentItem) []collection_dto.ContentItem {
	result := make([]collection_dto.ContentItem, len(itemPtrs))
	for i, item := range itemPtrs {
		result[i] = *item
	}
	return result
}

// applyLocaleFilter filters items by locale if one is given.
//
// Takes items ([]*collection_dto.ContentItem) which is the list to filter.
// Takes locale (string) which is the locale to match, or empty to skip
// filtering.
//
// Returns []*collection_dto.ContentItem which contains only items that match
// the locale, or all items if locale is empty.
func applyLocaleFilter(items []*collection_dto.ContentItem, locale string) []*collection_dto.ContentItem {
	if locale == "" {
		return items
	}

	filtered := make([]*collection_dto.ContentItem, 0, len(items))
	for _, item := range items {
		if item.Locale == locale {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// applyCustomFilters applies custom filter groups to content items.
//
// Takes items ([]*collection_dto.ContentItem) which is the list to filter.
// Takes filters (map[string]any) which may contain a "_filterGroup" key.
//
// Returns []*collection_dto.ContentItem which contains items that match the
// filter group, or the original items if no filter group is found.
func applyCustomFilters(items []*collection_dto.ContentItem, filters map[string]any) []*collection_dto.ContentItem {
	filterGroupData, ok := filters["_filterGroup"]
	if !ok {
		return items
	}

	filterGroup, ok := filterGroupData.(*collection_dto.FilterGroup)
	if !ok {
		return items
	}

	filtered := make([]*collection_dto.ContentItem, 0, len(items))
	for _, item := range items {
		if collection_dto.ApplyFilterGroup(item, filterGroup) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// applySorting sorts items based on options found in the filters map.
//
// Takes items ([]*collection_dto.ContentItem) which are the items to sort.
// Takes filters (map[string]any) which may contain a "_sort" key with sort
// options.
func applySorting(items []*collection_dto.ContentItem, filters map[string]any) {
	sortData, ok := filters["_sort"]
	if !ok {
		return
	}

	sortOptions, ok := sortData.([]collection_dto.SortOption)
	if !ok || len(sortOptions) == 0 {
		return
	}

	collection_dto.SortItems(items, sortOptions)
}

// applyPagination applies pagination to items if set in the filters.
//
// Takes items ([]*collection_dto.ContentItem) which is the list to paginate.
// Takes filters (map[string]any) which may contain a "_pagination" key.
//
// Returns []*collection_dto.ContentItem which is the paginated subset, or the
// original items if no pagination is set.
func applyPagination(items []*collection_dto.ContentItem, filters map[string]any) []*collection_dto.ContentItem {
	paginationData, ok := filters["_pagination"]
	if !ok {
		return items
	}

	pagination, ok := paginationData.(*collection_dto.PaginationOptions)
	if !ok {
		return items
	}

	return collection_dto.PaginateItems(items, pagination)
}
