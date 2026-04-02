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
	"github.com/stretchr/testify/require"
)

func TestNewNode(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		vector     []float32
		level      int
		wantLevel  int
		wantLayers int
	}{
		{
			name:       "base layer only",
			key:        "a",
			vector:     []float32{1, 0, 0},
			level:      0,
			wantLevel:  0,
			wantLayers: 1,
		},
		{
			name:       "multiple layers",
			key:        "b",
			vector:     []float32{0, 1, 0},
			level:      3,
			wantLevel:  3,
			wantLayers: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := newNode(tt.key, 0, tt.vector, tt.level, 16, 32)

			assert.Equal(t, tt.key, n.key)
			assert.Equal(t, tt.vector, n.vector)
			assert.Equal(t, tt.wantLevel, n.level)
			require.Len(t, n.neighbours, tt.wantLayers)

			for i := range n.neighbours {
				assert.NotNil(t, n.neighbours[i], "layer %d should have an initialised map", i)
				assert.Empty(t, n.neighbours[i], "layer %d should be empty initially", i)
			}
		})
	}
}

func TestMaxNeighbours(t *testing.T) {
	tests := []struct {
		name                   string
		layer                  int
		maxNeighboursPerLayer  int
		maxNeighboursBaseLayer int
		want                   int
	}{
		{name: "base layer", layer: 0, maxNeighboursPerLayer: 16, maxNeighboursBaseLayer: 32, want: 32},
		{name: "upper layer 1", layer: 1, maxNeighboursPerLayer: 16, maxNeighboursBaseLayer: 32, want: 16},
		{name: "upper layer 5", layer: 5, maxNeighboursPerLayer: 8, maxNeighboursBaseLayer: 16, want: 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, maxNeighbours(tt.layer, tt.maxNeighboursPerLayer, tt.maxNeighboursBaseLayer))
		})
	}
}
