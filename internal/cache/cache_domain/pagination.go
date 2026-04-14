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

const (
	// DefaultSearchLimit is the default number of results to return when no
	// limit is specified.
	DefaultSearchLimit = 10
)

// ApplyPagination applies offset and limit to a slice, returning the paginated
// slice and the resolved limit.
//
// Takes items ([]T) which is the slice to paginate.
// Takes offset (int) which specifies the number of items to skip.
// Takes limit (int) which specifies the maximum number of items to return.
//
// Returns []T which is the paginated slice.
// Returns int which is the resolved limit after applying constraints.
func ApplyPagination[T any](items []T, offset, limit int) ([]T, int) {
	if offset > 0 {
		if offset >= len(items) {
			items = nil
		} else {
			items = items[offset:]
		}
	}

	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	if limit > len(items) {
		limit = len(items)
	}
	items = items[:limit]

	return items, limit
}
