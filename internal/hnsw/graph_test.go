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

package hnsw

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/vectormaths"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name                           string
		opts                           []Option
		wantMaxNeighboursPerLayer      int
		wantMaxNeighboursBaseLayer     int
		wantConstructionCandidateCount int
		wantSearchCandidateCount       int
	}{
		{
			name:                           "defaults",
			wantMaxNeighboursPerLayer:      defaultMaxNeighboursPerLayer,
			wantMaxNeighboursBaseLayer:     defaultMaxNeighboursPerLayer * 2,
			wantConstructionCandidateCount: defaultConstructionCandidateCount,
			wantSearchCandidateCount:       defaultSearchCandidateCount,
		},
		{
			name:                           "custom values",
			opts:                           []Option{WithMaxNeighboursPerLayer(32), WithConstructionCandidateCount(400), WithSearchCandidateCount(100), WithRandomSeed(123)},
			wantMaxNeighboursPerLayer:      32,
			wantMaxNeighboursBaseLayer:     64,
			wantConstructionCandidateCount: 400,
			wantSearchCandidateCount:       100,
		},
		{
			name:                           "invalid values use defaults",
			opts:                           []Option{WithMaxNeighboursPerLayer(-1), WithConstructionCandidateCount(0), WithSearchCandidateCount(-5)},
			wantMaxNeighboursPerLayer:      defaultMaxNeighboursPerLayer,
			wantMaxNeighboursBaseLayer:     defaultMaxNeighboursPerLayer * 2,
			wantConstructionCandidateCount: defaultConstructionCandidateCount,
			wantSearchCandidateCount:       defaultSearchCandidateCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](3, vectormaths.Cosine, tt.opts...)

			assert.Equal(t, tt.wantMaxNeighboursPerLayer, g.maxNeighboursPerLayer)
			assert.Equal(t, tt.wantMaxNeighboursBaseLayer, g.maxNeighboursBaseLayer)
			assert.Equal(t, tt.wantConstructionCandidateCount, g.constructionCandidateCount)
			assert.Equal(t, tt.wantSearchCandidateCount, g.searchCandidateCount)
		})
	}
}

func TestGraph_Len(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))
	assert.Equal(t, 0, g.Len())

	g.Insert("a", []float32{1, 0, 0})
	assert.Equal(t, 1, g.Len())

	g.Insert("b", []float32{0, 1, 0})
	assert.Equal(t, 2, g.Len())
}

func TestGraph_Clear(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	g.Insert("a", []float32{1, 0, 0})
	g.Insert("b", []float32{0, 1, 0})
	g.Clear()

	assert.Equal(t, 0, g.Len())

	results := g.Search([]float32{1, 0, 0}, 1, 0)
	assert.Nil(t, results)

	g.Insert("c", []float32{0, 0, 1})
	assert.Equal(t, 1, g.Len())
}
