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
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_adapters/provider_otter"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/cache/cache_dto"
	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

var registerVectorDocBlueprint sync.Once

func TestLLM_Vertical_HNSW_RAG_Flow(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	const dimension = 4
	const blueprintName = "vector-doc-test-blueprint"

	otterProvider := provider_otter.NewOtterProvider()
	cacheService := cache_domain.NewService("otter")
	require.NoError(t, cacheService.RegisterProvider(context.Background(), "otter", otterProvider))

	registerVectorDocBlueprint.Do(func() {
		cache_domain.RegisterProviderFactory(blueprintName, func(service cache_domain.Service, namespace string, options any) (any, error) {
			opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
			if !ok {
				return nil, errors.New("invalid options type for vector doc blueprint")
			}
			return provider_otter.OtterProviderFactory[string, llm_dto.VectorDocument](opts)
		})
	})

	vStore := vector_cache.New(func(ns string, config *llm_domain.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
		return cache_domain.NewCacheBuilder[string, llm_dto.VectorDocument](cacheService).
			FactoryBlueprint(blueprintName).
			Namespace(ns).
			Searchable(cache_dto.NewSearchSchema(
				cache_dto.VectorFieldWithMetric("Vector", dimension, string(config.Metric)),
			)).
			Build(context.Background())
	})
	h.service.SetVectorStore(vStore)

	err := vStore.CreateNamespace(ctx, "kb", &llm_domain.VectorNamespaceConfig{
		Dimension: dimension,
		Metric:    llm_dto.SimilarityCosine,
	})
	require.NoError(t, err)

	targetVector := []float32{1.0, 0.0, 0.0, 0.0}
	err = vStore.Store(ctx, "kb", &llm_dto.VectorDocument{
		ID:      "piko-doc",
		Content: "Piko uses HNSW for fast vector search.",
		Vector:  targetVector,
	})
	require.NoError(t, err)

	err = vStore.Store(ctx, "kb", &llm_dto.VectorDocument{
		ID:      "noise-doc",
		Content: "Irrelevant content about weather.",
		Vector:  []float32{0.0, 0.0, 0.0, 1.0},
	})
	require.NoError(t, err)

	searchResp, err := vStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "kb",
		Vector:    []float32{0.9, 0.1, 0.0, 0.0},
		TopK:      1,
	})
	require.NoError(t, err)
	require.True(t, searchResp.HasResults())
	assert.Equal(t, "piko-doc", searchResp.Results[0].ID)

	h.mockProvider.SetResponse(makeResponse("I found information about Piko's search.", 10, 10))

	llmResp, err := h.service.NewCompletion().
		Model("test-model").
		User("How does Piko search?").
		WithVectorContext(searchResp.Results).
		Do(ctx)

	require.NoError(t, err)
	assert.NotNil(t, llmResp)

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 1)

	assert.Contains(t, calls[0].Messages[0].Content, "Piko uses HNSW for fast vector search.")
}
