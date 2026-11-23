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
	"math"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestGeneratedPikoScenario(t *testing.T) {
	raw, err := hex.DecodeString(os.Getenv("PIKO_FUZZ_DRAWS"))
	require.NoError(t, err)
	for _, metric := range []llm_dto.SimilarityMetric{llm_dto.SimilarityCosine, llm_dto.SimilarityEuclidean, llm_dto.SimilarityDotProduct} {
		store := newTestStore(t)
		const ns = "catalog"
		require.NoError(t, store.CreateNamespace(t.Context(), ns, &llm_domain.VectorNamespaceConfig{Metric: metric, Dimension: 3}))
		docs := map[string]llm_dto.VectorDocument{}
		steps := append([]byte{0, 3, 6, 9, 12, 1, 8, 7}, raw[:min(12, len(raw))]...)
		for step, b := range steps {
			id := fmt.Sprintf("item-%d", (b/3)%5)
			if b%3 == 2 {
				require.NoError(t, store.Delete(t.Context(), ns, id))
				delete(docs, id)
			} else {
				vec := []float32{float32(int(b)%7 - 3), float32((step+int(b))%9 - 4), float32(1 + int(b)%4)}
				doc := llm_dto.VectorDocument{ID: id, Content: fmt.Sprintf("revision %d", step), Vector: vec, Metadata: map[string]any{"group": step % 2}}
				require.NoError(t, store.Store(t.Context(), ns, &doc))
				docs[id] = doc
			}
			req := &llm_dto.VectorSearchRequest{Namespace: ns, TopK: 1 + int(b%5), Vector: []float32{float32(step%3 - 1), float32(int(b)%5 - 2), 2}, IncludeMetadata: b%2 == 0, IncludeVectors: step%2 == 0}
			if step%3 == 2 {
				req.MinScore = new(float32(0.37))
			}
			if step%4 == 3 {
				req.Filter = map[string]any{"group": step % 2}
				req.TopK = 5
			}
			response, err := store.Search(t.Context(), req)
			require.NoError(t, err)
			scores := map[string]float64{}
			var ranked []float64
			for id, doc := range docs {
				if req.Filter != nil && req.Filter["group"] != doc.Metadata["group"] {
					continue
				}
				dot, a2, b2, d2 := 0., 0., 0., 0.
				for j, v := range doc.Vector {
					x, y := float64(v), float64(req.Vector[j])
					dot += x * y
					a2 += x * x
					b2 += y * y
					d2 += (x - y) * (x - y)
				}
				score := dot / math.Sqrt(a2*b2)
				if metric == llm_dto.SimilarityDotProduct {
					score = dot
				}
				if metric == llm_dto.SimilarityEuclidean {
					score = 1 / (1 + math.Sqrt(d2))
				}
				if req.MinScore != nil && score < float64(*req.MinScore) {
					continue
				}
				scores[id] = score
				ranked = append(ranked, score)
			}
			slices.Sort(ranked)
			slices.Reverse(ranked)
			ranked = ranked[:min(len(ranked), req.TopK)]
			require.Len(t, response.Results, len(ranked), "PIKO_CHECK vector-count: metric %s step %d", metric, step)
			seen := map[string]bool{}
			for i, hit := range response.Results {
				expected, ok := scores[hit.ID]
				require.True(t, ok)
				require.False(t, seen[hit.ID])
				seen[hit.ID] = true
				require.InDelta(t, ranked[i], expected, 0.00001, "PIKO_CHECK vector-rank: metric %s step %d", metric, step)
				require.InDelta(t, expected, hit.Score, 0.00001, "PIKO_CHECK vector-score: metric %s step %d", metric, step)
				doc := docs[hit.ID]
				require.Equal(t, doc.Content, hit.Content)
				if req.IncludeMetadata {
					require.Equal(t, doc.Metadata, hit.Metadata)
				} else {
					require.Nil(t, hit.Metadata)
				}
				if req.IncludeVectors {
					require.Equal(t, doc.Vector, hit.Vector)
				} else {
					require.Nil(t, hit.Vector)
				}
			}
		}
		require.ErrorIs(t, store.Store(t.Context(), ns, &llm_dto.VectorDocument{ID: "invalid", Vector: []float32{1}}), llm_domain.ErrInvalidVector)
		v, err := store.Get(t.Context(), ns, "invalid")
		require.NoError(t, err)
		require.Nil(t, v, "PIKO_CHECK vector-invalid: rejected store did not publish")
	}
}
