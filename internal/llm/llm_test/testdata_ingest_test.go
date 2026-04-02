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
	"strings"
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

var registerTestdataDocBlueprint sync.Once

func TestLLM_TestData_Ingest_And_Query(t *testing.T) {
	h := newTestHarness(t)
	ctx := context.Background()

	const dimension = 6
	const blueprintName = "testdata-vector-doc-blueprint"

	otterProvider := provider_otter.NewOtterProvider()
	cacheService := cache_domain.NewService("otter")
	require.NoError(t, cacheService.RegisterProvider(context.Background(), "otter", otterProvider))

	registerTestdataDocBlueprint.Do(func() {
		cache_domain.RegisterProviderFactory(blueprintName, func(service cache_domain.Service, namespace string, options any) (any, error) {
			opts, ok := options.(cache_dto.Options[string, llm_dto.VectorDocument])
			if !ok {
				return nil, errors.New("invalid options type")
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

	err := vStore.CreateNamespace(ctx, "knowledge-base", &llm_domain.VectorNamespaceConfig{
		Dimension: dimension,
		Metric:    llm_dto.SimilarityCosine,
	})
	require.NoError(t, err)

	mockEmb := llm_domain.NewMockEmbeddingProvider()
	mockEmb.SetEmbedFunc(func(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		embeddings := make([]llm_dto.Embedding, len(request.Input))
		for i, text := range request.Input {
			vec := make([]float32, dimension)
			tLower := strings.ToLower(text)

			switch {
			case strings.Contains(tLower, "piko"):
				vec[0] = 1.0
			case strings.Contains(tLower, "pasta") || strings.Contains(tLower, "cook"):
				vec[1] = 1.0
			case strings.Contains(tLower, "mars") || strings.Contains(tLower, "space"):
				vec[2] = 1.0
			case strings.Contains(tLower, "plant") || strings.Contains(tLower, "garden"):
				vec[3] = 1.0
			case strings.Contains(tLower, "security") || strings.Contains(tLower, "secure"):
				vec[4] = 1.0
			case strings.Contains(tLower, "concur") || strings.Contains(tLower, "goroutine"):
				vec[5] = 1.0
			default:

				for j := range vec {
					vec[j] = 0.1
				}
			}
			embeddings[i] = llm_dto.Embedding{Index: i, Vector: vec}
		}

		return &llm_dto.EmbeddingResponse{Embeddings: embeddings}, nil
	})
	h.RegisterEmbeddingProvider("smart-emb", mockEmb)
	require.NoError(t, h.service.SetDefaultEmbeddingProvider("smart-emb"))

	splitter, splitterErr := llm_domain.NewRecursiveCharacterSplitter(500, 50)
	require.NoError(t, splitterErr)

	err = h.service.NewIngest("knowledge-base").
		FromDirectory("./testdata", "*.md").
		Splitter(splitter).
		Do(ctx)
	require.NoError(t, err)

	testCases := []struct {
		query          string
		expectedSource string
		expectedTerm   string
	}{
		{query: "Tell me about Piko", expectedSource: "piko.md", expectedTerm: "HNSW"},
		{query: "How to grow plants?", expectedSource: "gardening.md", expectedTerm: "Monsteras"},
		{query: "Secure my web app", expectedSource: "security.md", expectedTerm: "JWT"},
		{query: "Go concurrency primitives", expectedSource: "concurrency.md", expectedTerm: "Goroutines"},
		{query: "Cooking dinner", expectedSource: "cooking.md", expectedTerm: "pasta"},
		{query: "Mission to Mars", expectedSource: "space.md", expectedTerm: "Crater"},
	}

	for _, tc := range testCases {
		t.Run(tc.query, func(t *testing.T) {
			embResp, err := h.service.NewEmbedding().Input(tc.query).Embed(ctx)
			require.NoError(t, err)

			searchResp, err := h.service.GetVectorStore().Search(ctx, &llm_dto.VectorSearchRequest{
				Namespace:       "knowledge-base",
				Vector:          embResp.FirstVector(),
				TopK:            1,
				IncludeMetadata: true,
			})
			require.NoError(t, err)
			require.True(t, searchResp.HasResults(), "Should have results for query: %s", tc.query)

			assert.Equal(t, tc.expectedSource, searchResp.Results[0].Metadata["source"], "Query: %s", tc.query)
			assert.Contains(t, searchResp.Results[0].Content, tc.expectedTerm, "Query: %s", tc.query)
		})
	}

	h.mockProvider.SetResponse(makeResponse("Verified contextual response.", 20, 20))

	finalQuery := "How does Piko search?"
	embResp, _ := h.service.NewEmbedding().Input(finalQuery).Embed(ctx)
	searchResp, _ := h.service.GetVectorStore().Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "knowledge-base",
		Vector:    embResp.FirstVector(),
		TopK:      1,
	})

	llmResp, err := h.service.NewCompletion().
		Model("test-model").
		User(finalQuery).
		WithVectorContext(searchResp.Results).
		Do(ctx)

	require.NoError(t, err)
	assert.NotNil(t, llmResp)

	calls := h.mockProvider.GetCompleteCalls()
	require.NotEmpty(t, calls)
	lastCall := calls[len(calls)-1]

	assert.Contains(t, lastCall.Messages[0].Content, "Piko is a high-performance website development kit")
	assert.Contains(t, lastCall.Messages[0].Content, "[Document 1")
}
