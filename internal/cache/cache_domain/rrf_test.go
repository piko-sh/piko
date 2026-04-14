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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeRRFScores_BothLists(t *testing.T) {
	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.95},
		{Key: "doc2", Score: 0.85},
	}
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.5},
		{Key: "doc3", Score: 1.8},
	}

	entries := ComputeRRFScores(vectorHits, textScored)

	require.Len(t, entries, 3, "should have 3 unique keys from union")
	assert.Equal(t, "doc1", entries[0].Key, "doc1 should be ranked first (in both lists)")
	assert.Greater(t, entries[0].Score, entries[1].Score,
		"doc1 score should be higher than items in only one list")
}

func TestComputeRRFScores_ItemInBothListsScoresHigher(t *testing.T) {
	vectorHits := []VectorHit[string]{
		{Key: "both", Score: 0.9},
		{Key: "vec_only", Score: 0.8},
	}
	textScored := []ScoredResult[string]{
		{Key: "both", Score: 2.0},
		{Key: "text_only", Score: 1.5},
	}

	entries := ComputeRRFScores(vectorHits, textScored)

	require.Len(t, entries, 3)
	assert.Equal(t, "both", entries[0].Key)

	keys := make(map[string]bool)
	for _, entry := range entries {
		keys[entry.Key] = true
	}
	assert.True(t, keys["vec_only"])
	assert.True(t, keys["text_only"])
}

func TestComputeRRFScores_PureVectorNoText(t *testing.T) {
	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.95},
		{Key: "doc2", Score: 0.85},
	}
	var textScored []ScoredResult[string]

	entries := ComputeRRFScores(vectorHits, textScored)

	assert.Len(t, entries, 2)
}

func TestComputeRRFScores_PureTextNoVector(t *testing.T) {
	var vectorHits []VectorHit[string]
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.5},
		{Key: "doc2", Score: 1.8},
	}

	entries := ComputeRRFScores(vectorHits, textScored)

	assert.Len(t, entries, 2)
}

func TestComputeRRFScores_ScoreCalculation(t *testing.T) {
	vectorHits := []VectorHit[string]{
		{Key: "doc1", Score: 0.9},
	}
	textScored := []ScoredResult[string]{
		{Key: "doc1", Score: 2.0},
	}

	entries := ComputeRRFScores(vectorHits, textScored)

	require.Len(t, entries, 1)

	expectedScore := 2.0 / float64(RRFK+1)
	assert.InDelta(t, expectedScore, entries[0].Score, 0.0001,
		"RRF score should be the sum of reciprocal ranks from both lists")
}
