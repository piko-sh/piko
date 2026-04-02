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

package provider_otter

import (
	"cmp"
	"slices"

	"piko.sh/piko/internal/cache/cache_dto"
)

const (
	// rrfK is the Reciprocal Rank Fusion constant from Cormack, Clarke,
	// Buettcher (2009), where higher values reduce the influence of
	// high-ranked items.
	rrfK = 60
)

// rrfFusion combines vector similarity results with text relevance results
// using Reciprocal Rank Fusion. Items appearing in both lists receive scores
// from both, producing a union rather than an intersection.
//
// Takes vectorHits ([]VectorHit[K]) which are vector search results sorted by
// similarity descending.
// Takes textScored ([]ScoredResult[K]) which are text search results sorted by
// BM25 score descending.
// Takes filters ([]cache_dto.Filter) which are additional metadata filters to
// apply.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns SearchResult with items scored by RRF and sorted descending.
func (a *OtterAdapter[K, V]) rrfFusion(
	vectorHits []VectorHit[K],
	textScored []ScoredResult[K],
	filters []cache_dto.Filter,
	offset, limit int,
) (cache_dto.SearchResult[K, V], error) {
	rrfScores := make(map[K]float64, len(vectorHits)+len(textScored))

	for rank, hit := range vectorHits {
		rrfScores[hit.Key] += 1.0 / float64(rrfK+rank+1)
	}

	for rank, scored := range textScored {
		rrfScores[scored.Key] += 1.0 / float64(rrfK+rank+1)
	}

	type rrfEntry struct {
		key   K
		score float64
	}

	entries := make([]rrfEntry, 0, len(rrfScores))
	for k, s := range rrfScores {
		entries = append(entries, rrfEntry{key: k, score: s})
	}

	slices.SortFunc(entries, func(a, b rrfEntry) int {
		return cmp.Compare(b.score, a.score)
	})

	items := make([]cache_dto.SearchHit[K, V], 0, len(entries))
	for _, entry := range entries {
		value, ok := a.client.GetIfPresent(entry.key)
		if !ok {
			continue
		}

		if len(filters) > 0 && a.fieldExtractor != nil && !a.matchesAllFilters(value, filters) {
			continue
		}

		items = append(items, cache_dto.SearchHit[K, V]{
			Key:   entry.key,
			Value: value,
			Score: entry.score,
		})
	}

	total := int64(len(items))
	items, limit = applyPagination(items, offset, limit)

	return cache_dto.SearchResult[K, V]{
		Items:  items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
