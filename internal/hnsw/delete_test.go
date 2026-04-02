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

	"piko.sh/piko/internal/vectormaths"
)

func TestGraph_Delete(t *testing.T) {
	tests := []struct {
		name       string
		inserts    map[string][]float32
		deleteKeys []string
		wantLen    int
	}{
		{
			name: "delete one of three",
			inserts: map[string][]float32{
				"a": {1, 0, 0},
				"b": {0, 1, 0},
				"c": {0, 0, 1},
			},
			deleteKeys: []string{"a"},
			wantLen:    2,
		},
		{
			name: "delete non-existent key",
			inserts: map[string][]float32{
				"a": {1, 0, 0},
			},
			deleteKeys: []string{"nonexistent"},
			wantLen:    1,
		},
		{
			name: "delete all",
			inserts: map[string][]float32{
				"a": {1, 0, 0},
				"b": {0, 1, 0},
			},
			deleteKeys: []string{"a", "b"},
			wantLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))
			for k, v := range tt.inserts {
				g.Insert(k, v)
			}

			for _, k := range tt.deleteKeys {
				g.Delete(k)
			}

			assert.Equal(t, tt.wantLen, g.Len())
		})
	}
}

func TestGraph_DeletedKeysNotInResults(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	g.Insert("a", []float32{1, 0, 0})
	g.Insert("b", []float32{0, 1, 0})
	g.Insert("c", []float32{0, 0, 1})
	g.Delete("a")

	results := g.Search([]float32{1, 0, 0}, 5, 0)
	for _, r := range results {
		assert.NotEqual(t, "a", r.Key, "deleted key should not appear in results")
	}
}

func TestGraph_DeleteAllThenSearch(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	g.Insert("a", []float32{1, 0, 0})
	g.Insert("b", []float32{0, 1, 0})
	g.Delete("a")
	g.Delete("b")

	results := g.Search([]float32{1, 0, 0}, 5, 0)
	assert.Nil(t, results)
}

func TestGraph_DeleteEntryPoint(t *testing.T) {
	g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

	g.Insert("first", []float32{1, 0, 0})
	g.Insert("second", []float32{0, 1, 0})
	g.Insert("third", []float32{0, 0, 1})

	g.mu.RLock()
	entryKey := g.entry
	g.mu.RUnlock()

	g.Delete(entryKey)
	assert.Equal(t, 2, g.Len())

	results := g.Search([]float32{0, 1, 0}, 1, 0)
	require.Len(t, results, 1)
}

func TestGraph_DeleteAndSearch(t *testing.T) {
	const n = 50
	g := New[int](3, vectormaths.Cosine, WithRandomSeed(42))

	for i := range n {
		g.Insert(i, []float32{float32(i), float32(n - i), 1})
	}

	for i := 0; i < n; i += 2 {
		g.Delete(i)
	}

	assert.Equal(t, n/2, g.Len())

	results := g.Search([]float32{25, 25, 1}, 5, 50)
	require.NotEmpty(t, results)

	for _, r := range results {
		assert.NotEqual(t, 0, r.Key%2, "even keys should have been deleted")
	}
}

func TestGraph_ElectNewEntry(t *testing.T) {
	t.Run("empty graph after last delete", func(t *testing.T) {
		g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))
		g.Insert("only", []float32{1, 0, 0})
		g.Delete("only")

		assert.Equal(t, 0, g.Len())
		assert.Nil(t, g.Search([]float32{1, 0, 0}, 1, 0))
	})

	t.Run("fallback to nodes map when layers empty", func(t *testing.T) {
		g := New[string](3, vectormaths.Cosine, WithRandomSeed(42))

		g.Insert("a", []float32{1, 0, 0})
		g.Insert("b", []float32{0, 1, 0})

		g.mu.Lock()

		for i := range g.layers {
			g.layers[i] = make(map[string]*node[string])
		}
		g.electNewEntry()
		g.mu.Unlock()

		assert.True(t, g.hasEntry)
	})
}
