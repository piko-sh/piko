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
	"cmp"
	"slices"
)

const (
	// RRFK is the Reciprocal Rank Fusion constant from Cormack, Clarke,
	// Buettcher, where higher values reduce the influence of
	// high-ranked items.
	RRFK = 60
)

// RRFEntry holds a key and its fused RRF score.
type RRFEntry[K comparable] struct {
	// Key is the cache key.
	Key K

	// Score is the fused RRF score.
	Score float64
}

// ComputeRRFScores combines vector similarity results with text relevance
// results using Reciprocal Rank Fusion. Items appearing in both lists receive
// scores from both, producing a union rather than an intersection.
//
// Takes vectorHits ([]VectorHit[K]) which are vector search results sorted by
// similarity descending.
// Takes textScored ([]ScoredResult[K]) which are text search results sorted by
// BM25 score descending.
//
// Returns []RRFEntry[K] sorted by RRF score descending.
func ComputeRRFScores[K comparable](vectorHits []VectorHit[K], textScored []ScoredResult[K]) []RRFEntry[K] {
	rrfScores := make(map[K]float64, len(vectorHits)+len(textScored))

	for rank, hit := range vectorHits {
		rrfScores[hit.Key] += 1.0 / float64(RRFK+rank+1)
	}

	for rank, scored := range textScored {
		rrfScores[scored.Key] += 1.0 / float64(RRFK+rank+1)
	}

	entries := make([]RRFEntry[K], 0, len(rrfScores))
	for k, s := range rrfScores {
		entries = append(entries, RRFEntry[K]{Key: k, Score: s})
	}

	slices.SortFunc(entries, func(a, b RRFEntry[K]) int {
		return cmp.Compare(b.Score, a.Score)
	})

	return entries
}
