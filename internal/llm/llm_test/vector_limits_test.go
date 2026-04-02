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

package llm_test

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func makeRandomVector(randomSource *rand.Rand, dim int) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = randomSource.Float32()*2 - 1
	}
	return v
}

func normaliseVector(v []float32) []float32 {
	var norm float64
	for _, value := range v {
		norm += float64(value) * float64(value)
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return v
	}
	result := make([]float32, len(v))
	for i, value := range v {
		result[i] = float32(float64(value) / norm)
	}
	return result
}

func TestVector_SearchAccuracy_SmallCorpus(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 128, "cosine")
	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))

	const dim = 128

	queryVec := normaliseVector(makeRandomVector(randomSource, dim))
	exactDoc := &llm_dto.VectorDocument{
		ID:      "exact-match",
		Content: "This is the exact match document",
		Vector:  queryVec,
	}
	require.NoError(t, store.Store(ctx, "test", exactDoc))

	for i := range 99 {
		document := &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("random-%d", i),
			Content: fmt.Sprintf("Random document %d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
		}
		require.NoError(t, store.Store(ctx, "test", document))
	}

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "test",
		Vector:    queryVec,
		TopK:      1,
	})
	require.NoError(t, err)
	require.True(t, response.HasResults())

	first := response.FirstResult()
	require.NotNil(t, first)
	assert.Equal(t, "exact-match", first.ID, "exact match should be ranked #1")
	assert.InDelta(t, 1.0, float64(first.Score), 0.001, "cosine similarity with itself should be ~1.0")
}

func TestVector_SearchAccuracy_MediumCorpus(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 128, "cosine")
	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(123, 123>>1|1))

	const dim = 128

	relevantDir := normaliseVector(makeRandomVector(randomSource, dim))

	relevantIDs := make(map[string]bool)
	for i := range 10 {

		v := make([]float32, dim)
		for j := range v {
			v[j] = relevantDir[j] + (randomSource.Float32()-0.5)*0.1
		}
		id := fmt.Sprintf("relevant-%d", i)
		relevantIDs[id] = true
		require.NoError(t, store.Store(ctx, "corpus", &llm_dto.VectorDocument{
			ID:      id,
			Content: fmt.Sprintf("Relevant document %d", i),
			Vector:  normaliseVector(v),
		}))
	}

	for i := range 990 {
		require.NoError(t, store.Store(ctx, "corpus", &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("noise-%d", i),
			Content: fmt.Sprintf("Noise document %d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
		}))
	}

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "corpus",
		Vector:    relevantDir,
		TopK:      20,
	})
	require.NoError(t, err)
	require.Len(t, response.Results, 20)

	var relevantInTop20 int
	for _, r := range response.Results {
		if relevantIDs[r.ID] {
			relevantInTop20++
		}
	}
	assert.Equal(t, 10, relevantInTop20,
		"all 10 relevant documents should appear in top-20 results")
}

func TestVector_FilterCombinations(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 4, "cosine")
	ctx := context.Background()

	vec := normaliseVector([]float32{1, 0, 0, 0})

	docs := []*llm_dto.VectorDocument{
		{ID: "d1", Vector: vec, Content: "A", Metadata: map[string]any{"env": "prod", "team": "alpha", "priority": "high"}},
		{ID: "d2", Vector: vec, Content: "B", Metadata: map[string]any{"env": "prod", "team": "beta", "priority": "low"}},
		{ID: "d3", Vector: vec, Content: "C", Metadata: map[string]any{"env": "staging", "team": "alpha", "priority": "high"}},
		{ID: "d4", Vector: vec, Content: "D", Metadata: map[string]any{"env": "staging", "team": "beta", "priority": "low"}},
	}
	require.NoError(t, store.BulkStore(ctx, "filtered", docs))

	testCases := []struct {
		name     string
		filter   map[string]any
		expected []string
	}{
		{name: "single key", filter: map[string]any{"env": "prod"}, expected: []string{"d1", "d2"}},
		{name: "two keys", filter: map[string]any{"env": "prod", "team": "alpha"}, expected: []string{"d1"}},
		{name: "three keys", filter: map[string]any{"env": "staging", "team": "beta", "priority": "low"}, expected: []string{"d4"}},
		{name: "no match", filter: map[string]any{"env": "dev"}, expected: []string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
				Namespace:       "filtered",
				Vector:          vec,
				TopK:            10,
				Filter:          tc.filter,
				IncludeMetadata: true,
			})
			require.NoError(t, err)

			var ids []string
			for _, r := range response.Results {
				ids = append(ids, r.ID)
			}
			assert.ElementsMatch(t, tc.expected, ids)
		})
	}
}

func TestVector_AllMetrics_Consistency(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 5, "cosine")
	ctx := context.Background()

	metrics := []llm_dto.SimilarityMetric{
		llm_dto.SimilarityCosine,
		llm_dto.SimilarityEuclidean,
		llm_dto.SimilarityDotProduct,
	}

	queryVec := normaliseVector([]float32{1, 0, 0, 0, 0})
	closestVec := normaliseVector([]float32{0.9, 0.1, 0, 0, 0})
	farthestVec := normaliseVector([]float32{0, 0, 0, 0, 1})

	for _, metric := range metrics {
		ns := string(metric)
		require.NoError(t, store.CreateNamespace(ctx, ns, &llm_domain.VectorNamespaceConfig{
			Metric: metric,
		}))

		for _, document := range []*llm_dto.VectorDocument{
			{ID: "closest", Vector: closestVec, Content: "Closest"},
			{ID: "farthest", Vector: farthestVec, Content: "Farthest"},
		} {
			require.NoError(t, store.Store(ctx, ns, document))
		}
	}

	for _, metric := range metrics {
		t.Run(string(metric), func(t *testing.T) {
			response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
				Namespace: string(metric),
				Vector:    queryVec,
				TopK:      2,
			})
			require.NoError(t, err)
			require.Len(t, response.Results, 2)
			assert.Equal(t, "closest", response.Results[0].ID,
				"closest document should rank #1 with %s metric", metric)
		})
	}
}

func TestVector_HighDimensional(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 1536, "cosine")
	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))

	const dim = 1536

	queryVec := normaliseVector(makeRandomVector(randomSource, dim))

	require.NoError(t, store.Store(ctx, "high-dim", &llm_dto.VectorDocument{
		ID:      "target",
		Vector:  queryVec,
		Content: "Target document",
	}))

	for i := range 50 {
		require.NoError(t, store.Store(ctx, "high-dim", &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("random-%d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
			Content: fmt.Sprintf("Random %d", i),
		}))
	}

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:      "high-dim",
		Vector:         queryVec,
		TopK:           1,
		IncludeVectors: true,
	})
	require.NoError(t, err)
	require.True(t, response.HasResults())

	first := response.FirstResult()
	require.NotNil(t, first)
	assert.Equal(t, "target", first.ID)
	assert.Len(t, first.Vector, dim, "returned vector should have correct dimensionality")
}

func TestVector_MinScoreThreshold(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 3, "cosine")
	ctx := context.Background()

	queryVec := normaliseVector([]float32{1, 0, 0})

	docs := []*llm_dto.VectorDocument{
		{ID: "parallel", Vector: normaliseVector([]float32{1, 0, 0}), Content: "Parallel"},
		{ID: "similar", Vector: normaliseVector([]float32{0.9, 0.1, 0}), Content: "Similar"},
		{ID: "orthogonal", Vector: normaliseVector([]float32{0, 1, 0}), Content: "Orthogonal"},
		{ID: "opposite", Vector: normaliseVector([]float32{-1, 0, 0}), Content: "Opposite"},
	}
	require.NoError(t, store.BulkStore(ctx, "threshold", docs))

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "threshold",
		Vector:    queryVec,
		TopK:      10,
		MinScore:  new(float32(0.5)),
	})
	require.NoError(t, err)

	assert.Len(t, response.Results, 2, "only documents with score >= 0.5 should be returned")
	for _, r := range response.Results {
		assert.GreaterOrEqual(t, r.Score, float32(0.5))
	}
}

func TestVector_NamespaceIsolation(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 3, "cosine")
	ctx := context.Background()

	vec := normaliseVector([]float32{1, 0, 0})

	require.NoError(t, store.Store(ctx, "ns-a", &llm_dto.VectorDocument{
		ID: "doc-a", Vector: vec, Content: "In namespace A",
	}))
	require.NoError(t, store.Store(ctx, "ns-b", &llm_dto.VectorDocument{
		ID: "doc-b", Vector: vec, Content: "In namespace B",
	}))

	respA, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "ns-a",
		Vector:    vec,
		TopK:      10,
	})
	require.NoError(t, err)
	require.Len(t, respA.Results, 1)
	assert.Equal(t, "doc-a", respA.Results[0].ID)

	respB, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "ns-b",
		Vector:    vec,
		TopK:      10,
	})
	require.NoError(t, err)
	require.Len(t, respB.Results, 1)
	assert.Equal(t, "doc-b", respB.Results[0].ID)

	respC, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "ns-c",
		Vector:    vec,
		TopK:      10,
	})
	require.NoError(t, err)
	assert.Empty(t, respC.Results)
}

func TestVector_BulkStoreAndSearch(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 64, "cosine")
	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(99, 99>>1|1))

	const dim = 64
	const count = 500

	docs := make([]*llm_dto.VectorDocument, count)
	for i := range docs {
		docs[i] = &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("bulk-%d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
			Content: fmt.Sprintf("Bulk document %d", i),
		}
	}
	require.NoError(t, store.BulkStore(ctx, "bulk", docs))

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "bulk",
		Vector:    normaliseVector(makeRandomVector(randomSource, dim)),
		TopK:      10,
	})
	require.NoError(t, err)
	assert.Len(t, response.Results, 10, "should return TopK results")
	assert.GreaterOrEqual(t, response.TotalCount, 10, "total count should reflect all matching docs")

	targetDoc := docs[250]
	response, err = store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "bulk",
		Vector:    targetDoc.Vector,
		TopK:      1,
	})
	require.NoError(t, err)
	require.True(t, response.HasResults())
	firstResult := response.FirstResult()
	require.NotNil(t, firstResult)
	assert.Equal(t, targetDoc.ID, firstResult.ID, "searching by own vector should return the document")
}

func TestVector_ConcurrentReadWrite(t *testing.T) {
	skipIfShort(t)

	store := createOtterVectorStore(t, 32, "cosine")
	ctx := context.Background()

	const dim = 32
	const writers = 5
	const readers = 5
	const docsPerWriter = 50

	var wg sync.WaitGroup

	for w := range writers {
		wg.Go(func() {
			randomSource := rand.New(rand.NewPCG(uint64(w), uint64(w>>1|1)))
			for i := range docsPerWriter {
				document := &llm_dto.VectorDocument{
					ID:      fmt.Sprintf("w%d-d%d", w, i),
					Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
					Content: fmt.Sprintf("Writer %d doc %d", w, i),
				}
				_ = store.Store(ctx, "concurrent", document)
			}
		})
	}

	for r := range readers {
		wg.Go(func() {
			randomSource := rand.New(rand.NewPCG(uint64(100+r), uint64((100+r)>>1|1)))
			for range 20 {
				_, _ = store.Search(ctx, &llm_dto.VectorSearchRequest{
					Namespace: "concurrent",
					Vector:    normaliseVector(makeRandomVector(randomSource, dim)),
					TopK:      5,
				})
			}
		})
	}

	wg.Wait()

	response, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "concurrent",
		Vector:    normaliseVector(makeRandomVector(rand.New(rand.NewPCG(999, 999>>1|1)), dim)),
		TopK:      1000,
	})
	require.NoError(t, err)
	assert.Equal(t, writers*docsPerWriter, response.TotalCount,
		"all documents from all writers should be stored")
}
