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

//go:build bench

package llm_test_bench

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sync/atomic"
	"testing"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

var benchBPCounter atomic.Int64

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

func newBenchVectorStore(b *testing.B, dimension int) *vector_cache.Store {
	b.Helper()

	bp := fmt.Sprintf("otter-bench-%d", benchBPCounter.Add(1))
	cache_domain.RegisterProviderFactory(bp, func(_ cache_domain.Service, _ string, options any) (any, error) {
		opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
		if !ok {
			return nil, errors.New("invalid options type")
		}
		return provider_otter.OtterProviderFactory[string, llm_dto.VectorDocument](opts)
	})

	cacheService := cache_domain.NewService("otter")

	return vector_cache.New(func(ns string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		d := dimension
		if config != nil && config.Dimension > 0 {
			d = config.Dimension
		}
		schema := cache_dto.NewSearchSchema(
			cache_dto.VectorFieldWithMetric("Vector", d, "cosine"),
			cache_dto.TextField("Content"),
		)
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(bp).
			Namespace(ns).
			MaximumSize(200000).
			Searchable(schema).
			Build(context.Background())
	})
}

func seedStore(b *testing.B, store *vector_cache.Store, ns string, n, dim int) {
	b.Helper()
	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))

	const batchSize = 500
	batch := make([]*llm_dto.VectorDocument, 0, batchSize)

	for i := range n {
		batch = append(batch, &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("doc-%d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
			Content: fmt.Sprintf("Document %d", i),
		})
		if len(batch) >= batchSize {
			if err := store.BulkStore(ctx, ns, batch); err != nil {
				b.Fatalf("seeding store: %v", err)
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := store.BulkStore(ctx, ns, batch); err != nil {
			b.Fatalf("seeding store: %v", err)
		}
	}
}

func benchmarkVectorSearch(b *testing.B, n, dim int) {
	b.Helper()

	store := newBenchVectorStore(b, dim)
	ns := fmt.Sprintf("bench-%d-%d", n, dim)
	seedStore(b, store, ns, n, dim)

	ctx := context.Background()
	randomSource := rand.New(rand.NewPCG(99, 99>>1|1))
	queryVec := normaliseVector(makeRandomVector(randomSource, dim))

	request := &llm_dto.VectorSearchRequest{
		Namespace: ns,
		Vector:    queryVec,
		TopK:      10,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := store.Search(ctx, request)
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkVectorSearch_100(b *testing.B) {
	benchmarkVectorSearch(b, 100, 128)
}

func BenchmarkVectorSearch_1K(b *testing.B) {
	benchmarkVectorSearch(b, 1_000, 128)
}

func BenchmarkVectorSearch_10K(b *testing.B) {
	benchmarkVectorSearch(b, 10_000, 128)
}

func BenchmarkVectorSearch_100K(b *testing.B) {
	benchmarkVectorSearch(b, 100_000, 128)
}

func BenchmarkVectorSearch_Dim128(b *testing.B) {
	benchmarkVectorSearch(b, 10_000, 128)
}

func BenchmarkVectorSearch_Dim768(b *testing.B) {
	benchmarkVectorSearch(b, 10_000, 768)
}

func BenchmarkVectorSearch_Dim1536(b *testing.B) {
	benchmarkVectorSearch(b, 10_000, 1536)
}

func BenchmarkVectorBulkStore(b *testing.B) {
	randomSource := rand.New(rand.NewPCG(42, 42>>1|1))
	const dim = 128
	const count = 10_000

	docs := make([]*llm_dto.VectorDocument, count)
	for i := range docs {
		docs[i] = &llm_dto.VectorDocument{
			ID:      fmt.Sprintf("doc-%d", i),
			Vector:  normaliseVector(makeRandomVector(randomSource, dim)),
			Content: fmt.Sprintf("Document %d", i),
		}
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		store := newBenchVectorStore(b, 128)
		if err := store.BulkStore(ctx, "bench", docs); err != nil {
			b.Fatalf("bulk store failed: %v", err)
		}
	}
}

func BenchmarkVectorConcurrentSearch(b *testing.B) {
	store := newBenchVectorStore(b, 128)
	const dim = 128
	seedStore(b, store, "concurrent", 10_000, dim)

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		randomSource := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		for pb.Next() {
			queryVec := normaliseVector(makeRandomVector(randomSource, dim))
			_, err := store.Search(ctx, &llm_dto.VectorSearchRequest{
				Namespace: "concurrent",
				Vector:    queryVec,
				TopK:      10,
			})
			if err != nil {
				b.Fatalf("search failed: %v", err)
			}
		}
	})
}
