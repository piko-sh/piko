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
	// rrfK is the Reciprocal Rank Fusion constant from Cormack, Clarke, Buettcher (2009),
	// where higher values reduce the influence of high-ranked items.
	rrfK = 60
)

// rrfFusion combines vector and text results using Reciprocal Rank Fusion.
//
// Takes vectorHits ([]VectorHit[K]) which are vector search results sorted by similarity
// descending.
// Takes textScored ([]ScoredResult[K]) which are text search results sorted by BM25 score
// descending.
// Takes filters ([]cache_dto.Filter) which are additional metadata filters to apply.
// Takes offset (int) which is the pagination offset.
// Takes limit (int) which is the maximum number of results.
//
// Returns cache_dto.SearchResult[K, V] which contains the filtered page in descending RRF
// order.
// Returns error which is nil.
func (a *OtterAdapter[K, V]) rrfFusion(
	vectorHits []VectorHit[K],
	textScored []ScoredResult[K],
	filters []cache_dto.Filter,
	offset, limit int,
) (cache_dto.SearchResult[K, V], error) {
	type rrfEntry struct {
		key       K
		score     float64
		bestRank  int
		firstSeen int
	}

	entries := make([]rrfEntry, 0, len(vectorHits)+len(textScored))
	positions := make(map[K]int, len(vectorHits)+len(textScored))
	add := func(key K, rank int) {
		position, seen := positions[key]
		if !seen {
			position = len(entries)
			positions[key] = position
			entries = append(entries, rrfEntry{key: key, score: 0, bestRank: rank, firstSeen: position})
		}
		entry := &entries[position]
		entry.score += 1.0 / float64(rrfK+rank+1)
		entry.bestRank = min(entry.bestRank, rank)
	}
	for rank, hit := range vectorHits {
		add(hit.Key, rank)
	}
	for rank, scored := range textScored {
		add(scored.Key, rank)
	}

	slices.SortFunc(entries, func(a, b rrfEntry) int {
		return cmp.Or(
			cmp.Compare(b.score, a.score),
			cmp.Compare(a.bestRank, b.bestRank),
			cmp.Compare(a.firstSeen, b.firstSeen),
		)
	})

	return a.rankedPage(len(entries), len(vectorHits), func(i int) (K, float64) {
		return entries[i].key, entries[i].score
	}, filters, offset, limit), nil
}
