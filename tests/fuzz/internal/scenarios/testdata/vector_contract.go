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

package vector_cache

import (
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestGeneratedPikoScenario(t *testing.T) {
	ctx := t.Context()
	ds, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	ds = append(ds, 0, 0)
	scale := float32(1 + int(ds[0])%5)
	store := newTestStore(t)
	createTestNamespace(t, store, "catalog", 2)
	docs := []*llm_dto.VectorDocument{{ID: "closest", Vector: []float32{scale, 0}, Metadata: map[string]any{"group": "near"}}, {ID: "eligible", Vector: []float32{0, scale}, Metadata: map[string]any{"group": "selected"}}}
	for i := range int(ds[1] % 3) {
		docs = append(docs, &llm_dto.VectorDocument{ID: fmt.Sprintf("other-%d", i), Vector: []float32{-scale, float32(i + 1)}, Metadata: map[string]any{"group": "other"}})
	}
	require.NoError(t, store.BulkStore(ctx, "catalog", docs))
	t.Run("count before limit", func(t *testing.T) {
		got, err := store.Search(ctx, &llm_dto.VectorSearchRequest{Namespace: "catalog", Vector: []float32{1, 0}, TopK: 1})
		require.NoError(t, err)
		require.Len(t, got.Results, 1)
		require.Equal(t, "closest", got.Results[0].ID)
		require.GreaterOrEqual(t, got.TotalCount, len(got.Results), "PIKO_CHECK vector-total-covers-results: unfiltered namespace with TopK=1")
		require.LessOrEqual(t, got.TotalCount, len(docs), "PIKO_CHECK vector-total-within-namespace: unfiltered namespace with TopK=1")
	})
	t.Run("rank after filtering", func(t *testing.T) {
		got, err := store.Search(ctx, &llm_dto.VectorSearchRequest{Namespace: "catalog", Vector: []float32{1, 0}, TopK: 1, Filter: map[string]any{"group": "selected"}})
		require.NoError(t, err)
		require.Len(t, got.Results, 1, "PIKO_CHECK vector-filter-before-limit: one eligible document behind closer excluded document")
		require.Equal(t, "eligible", got.Results[0].ID)
		require.Equal(t, 1, got.TotalCount, "PIKO_CHECK vector-filtered-total: one eligible document")
	})
}
