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

package llm_domain

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
)

type ragMockVectorStore struct {
	searchFunc func(ctx context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error)
}

func (m *ragMockVectorStore) Store(_ context.Context, _ string, _ *llm_dto.VectorDocument) error {
	return nil
}

func (m *ragMockVectorStore) BulkStore(_ context.Context, _ string, _ []*llm_dto.VectorDocument) error {
	return nil
}

func (m *ragMockVectorStore) Search(ctx context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, request)
	}
	return &llm_dto.VectorSearchResponse{}, nil
}

func (m *ragMockVectorStore) Get(_ context.Context, _, _ string) (*llm_dto.VectorDocument, error) {
	return nil, nil
}

func (m *ragMockVectorStore) Delete(_ context.Context, _, _ string) error {
	return nil
}

func (m *ragMockVectorStore) DeleteByFilter(_ context.Context, _ string, _ map[string]any) (int, error) {
	return 0, nil
}

func (m *ragMockVectorStore) CreateNamespace(_ context.Context, _ string, _ *VectorNamespaceConfig) error {
	return nil
}

func (m *ragMockVectorStore) DeleteNamespace(_ context.Context, _ string) error {
	return nil
}

func (m *ragMockVectorStore) Close(_ context.Context) error {
	return nil
}

func TestRAG(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion().RAG("ns", 5)

	require.NotNil(t, b.ragConfig, "ragConfig should be set after calling RAG")
	assert.Equal(t, "ns", b.ragConfig.namespace)
	assert.Equal(t, 5, b.ragConfig.topK)
	assert.Empty(t, b.ragConfig.query, "query should be empty by default")
	assert.Empty(t, b.ragConfig.embeddingProvider, "embeddingProvider should be empty by default")
	assert.Empty(t, b.ragConfig.embeddingModel, "embeddingModel should be empty by default")
	assert.Nil(t, b.ragConfig.minScore, "minScore should be nil by default")
	assert.Nil(t, b.ragConfig.filter, "filter should be nil by default")
}

func TestRAGOptions(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	filter := map[string]any{"k": "v"}
	b := service.NewCompletion().RAG("ns", 5,
		WithRAGQuery("q"),
		WithRAGMinScore(0.8),
		WithRAGFilter(filter),
		WithRAGEmbeddingProvider("ep"),
		WithRAGEmbeddingModel("em"),
	)

	require.NotNil(t, b.ragConfig)
	assert.Equal(t, "ns", b.ragConfig.namespace)
	assert.Equal(t, 5, b.ragConfig.topK)
	assert.Equal(t, "q", b.ragConfig.query)
	require.NotNil(t, b.ragConfig.minScore)
	assert.InDelta(t, float32(0.8), *b.ragConfig.minScore, 0.001)
	assert.Equal(t, filter, b.ragConfig.filter)
	assert.Equal(t, "ep", b.ragConfig.embeddingProvider)
	assert.Equal(t, "em", b.ragConfig.embeddingModel)
}

func TestLastUserMessageContent(t *testing.T) {
	t.Run("returns last user message when multiple messages exist", func(t *testing.T) {
		service := NewService("mock")
		provider := NewMockLLMProvider()
		require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

		b := service.NewCompletion().
			System("system prompt").
			User("first").
			Assistant("reply").
			User("second")

		got := b.lastUserMessageContent()
		assert.Equal(t, "second", got)
	})

	t.Run("returns empty string when no messages", func(t *testing.T) {
		service := NewService("mock")
		provider := NewMockLLMProvider()
		require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

		b := service.NewCompletion()

		got := b.lastUserMessageContent()
		assert.Empty(t, got)
	})

	t.Run("returns empty string when no user messages", func(t *testing.T) {
		service := NewService("mock")
		provider := NewMockLLMProvider()
		require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

		b := service.NewCompletion().
			System("system prompt").
			Assistant("reply")

		got := b.lastUserMessageContent()
		assert.Empty(t, got)
	})
}

func newRAGTestService(t *testing.T, vs *ragMockVectorStore) (Service, *MockEmbeddingProvider) {
	t.Helper()

	service := NewService("mock")

	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	embProvider := NewMockEmbeddingProvider()
	require.NoError(t, service.RegisterEmbeddingProvider(context.Background(), "mock-embed", embProvider))
	require.NoError(t, service.SetDefaultEmbeddingProvider("mock-embed"))

	if vs != nil {
		service.SetVectorStore(vs)
	}

	return service, embProvider
}

func TestResolveRAGContext_Success(t *testing.T) {
	expectedVector := []float32{0.1, 0.2, 0.3}
	expectedResults := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Content: "first document", Score: 0.95},
		{ID: "doc-2", Content: "second document", Score: 0.85},
	}

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			assert.Equal(t, "ns", request.Namespace)
			assert.Equal(t, 3, request.TopK)
			assert.Equal(t, expectedVector, request.Vector)
			assert.True(t, request.IncludeMetadata)
			return &llm_dto.VectorSearchResponse{
				Results:    expectedResults,
				TotalCount: 2,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		assert.Equal(t, []string{"find me docs"}, request.Input)
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: expectedVector},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, b.vectorContext, "vectorContext should be populated after successful RAG resolution")
	assert.Len(t, b.vectorContext, 2)
	assert.Equal(t, "doc-1", b.vectorContext[0].ID)
	assert.Equal(t, "first document", b.vectorContext[0].Content)
	assert.InDelta(t, float32(0.95), b.vectorContext[0].Score, 0.001)
	assert.Equal(t, "doc-2", b.vectorContext[1].ID)
	assert.Equal(t, "second document", b.vectorContext[1].Content)
	assert.InDelta(t, float32(0.85), b.vectorContext[1].Score, 0.001)
}

func TestResolveRAGContext_ExplicitQuery(t *testing.T) {
	var capturedInput []string

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "result", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		capturedInput = request.Input
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{0.5, 0.6, 0.7}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3, WithRAGQuery("custom query")).
		User("this should not be used as query")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, capturedInput, "embedding should have been called")
	assert.Equal(t, []string{"custom query"}, capturedInput,
		"the explicit query should be used for embedding, not the user message")
	assert.NotNil(t, b.vectorContext, "vectorContext should be populated")
}

func TestResolveRAGContext_NoQuery(t *testing.T) {
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			t.Fatal("search should not be called when there is no query")
			return nil, nil
		},
	}

	service, _ := newRAGTestService(t, vs)

	b := service.NewCompletion().
		RAG("ns", 3).
		System("system only")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext, "vectorContext should be nil when no query text is available")
}

func TestResolveRAGContext_EmbeddingFails(t *testing.T) {
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			t.Fatal("search should not be called when embedding fails")
			return nil, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return nil, fmt.Errorf("embedding service unavailable")
	})

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext, "vectorContext should be nil when embedding fails")
}

func TestResolveRAGContext_NoResults(t *testing.T) {
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			return &llm_dto.VectorSearchResponse{
				Results:    []llm_dto.VectorSearchResult{},
				TotalCount: 0,
			}, nil
		},
	}

	service, _ := newRAGTestService(t, vs)

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext, "vectorContext should be nil when no results are returned")
}

func TestResolveRAGContext_NoVectorStore(t *testing.T) {

	service, _ := newRAGTestService(t, nil)

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext, "vectorContext should be nil when no vector store is configured")
}

func TestResolveRAGContext_NoEmbeddingProvider(t *testing.T) {

	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			t.Fatal("search should not be called when embedding provider is missing")
			return nil, nil
		},
	}
	service.SetVectorStore(vs)

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext,
		"vectorContext should be nil when no embedding provider is registered")
}

func TestResolveRAGContext_SearchError(t *testing.T) {
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			return nil, fmt.Errorf("vector store connection refused")
		},
	}

	service, _ := newRAGTestService(t, vs)

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext,
		"vectorContext should be nil when vector search returns an error")
}

func TestResolveRAGContext_EmptyEmbedding(t *testing.T) {
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			t.Fatal("search should not be called when embedding is empty")
			return nil, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3).
		User("find me docs")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext,
		"vectorContext should be nil when embedding returns empty vector")
}

func TestResolveRAGContext_NoRagConfig(t *testing.T) {
	service, _ := newRAGTestService(t, &ragMockVectorStore{})

	b := service.NewCompletion().
		User("no rag configured")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Nil(t, b.vectorContext,
		"vectorContext should be nil when ragConfig is not set")
	assert.Nil(t, b.ragConfig, "ragConfig should remain nil")
}

func TestResolveRAGContext_PassesOptionsToSearch(t *testing.T) {
	minScore := float32(0.7)
	filter := map[string]any{"category": "science"}

	var capturedReq *llm_dto.VectorSearchRequest
	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			capturedReq = request
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "science doc", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, _ := newRAGTestService(t, vs)

	b := service.NewCompletion().
		RAG("custom-ns", 10,
			WithRAGMinScore(minScore),
			WithRAGFilter(filter),
		).
		User("science question")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, capturedReq, "search should have been called")
	assert.Equal(t, "custom-ns", capturedReq.Namespace)
	assert.Equal(t, 10, capturedReq.TopK)
	require.NotNil(t, capturedReq.MinScore)
	assert.InDelta(t, float32(0.7), *capturedReq.MinScore, 0.001)
	assert.Equal(t, filter, capturedReq.Filter)
	assert.True(t, capturedReq.IncludeMetadata)
}

func TestResolveRAGContext_PassesEmbeddingModel(t *testing.T) {
	var capturedModel string

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "result", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		capturedModel = request.Model
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{0.1, 0.2}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3,
			WithRAGEmbeddingModel("text-embedding-3-large"),
		).
		User("query")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	assert.Equal(t, "text-embedding-3-large", capturedModel,
		"the embedding model from RAG options should be passed to the embedding request")
}

func TestRAG_DoPopulatesSources(t *testing.T) {
	expectedResults := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Content: "first document", Score: 0.95},
		{ID: "doc-2", Content: "second document", Score: 0.85},
	}

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, _ *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			return &llm_dto.VectorSearchResponse{
				Results:    expectedResults,
				TotalCount: 2,
			}, nil
		},
	}

	service, _ := newRAGTestService(t, vs)

	response, err := service.NewCompletion().
		Model("mock-model").
		RAG("ns", 3).
		User("find me docs").
		Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Sources, 2)
	assert.Equal(t, "doc-1", response.Sources[0].ID)
	assert.Equal(t, "first document", response.Sources[0].Content)
	assert.InDelta(t, float32(0.95), response.Sources[0].Score, 0.001)
	assert.Equal(t, "doc-2", response.Sources[1].ID)
}

func TestRAG_DoWithoutRAGHasNilSources(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	response, err := service.NewCompletion().
		Model("mock-model").
		User("no rag here").
		Do(context.Background())

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Nil(t, response.Sources)
}

func TestWithRAGHybridSearch_SetsFlag(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion().RAG("ns", 5, WithRAGHybridSearch())

	require.NotNil(t, b.ragConfig)
	assert.True(t, b.ragConfig.enableHybridSearch,
		"enableHybridSearch should be true after WithRAGHybridSearch()")
}

func TestWithRAGHybridSearch_DefaultDisabled(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	b := service.NewCompletion().RAG("ns", 5)

	require.NotNil(t, b.ragConfig)
	assert.False(t, b.ragConfig.enableHybridSearch,
		"enableHybridSearch should be false by default")
}

func TestResolveRAGContext_HybridSearchPassesTextQuery(t *testing.T) {
	var capturedReq *llm_dto.VectorSearchRequest

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			capturedReq = request
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "result", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{0.1, 0.2, 0.3}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3, WithRAGHybridSearch()).
		User("search query text")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, capturedReq, "search should have been called")
	assert.Equal(t, "search query text", capturedReq.TextQuery,
		"TextQuery should be set to the user message when hybrid search is enabled")
}

func TestResolveRAGContext_WithoutHybridSearchNoTextQuery(t *testing.T) {
	var capturedReq *llm_dto.VectorSearchRequest

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			capturedReq = request
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "result", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{0.1, 0.2, 0.3}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3).
		User("search query text")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, capturedReq, "search should have been called")
	assert.Empty(t, capturedReq.TextQuery,
		"TextQuery should be empty when hybrid search is not enabled (backward compatible)")
}

func TestResolveRAGContext_HybridSearchWithExplicitQuery(t *testing.T) {
	var capturedReq *llm_dto.VectorSearchRequest

	vs := &ragMockVectorStore{
		searchFunc: func(_ context.Context, request *llm_dto.VectorSearchRequest) (*llm_dto.VectorSearchResponse, error) {
			capturedReq = request
			return &llm_dto.VectorSearchResponse{
				Results: []llm_dto.VectorSearchResult{
					{ID: "doc-1", Content: "result", Score: 0.9},
				},
				TotalCount: 1,
			}, nil
		},
	}

	service, embProvider := newRAGTestService(t, vs)
	embProvider.SetEmbedFunc(func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return &llm_dto.EmbeddingResponse{
			Embeddings: []llm_dto.Embedding{
				{Index: 0, Vector: []float32{0.1, 0.2, 0.3}},
			},
		}, nil
	})

	b := service.NewCompletion().
		RAG("ns", 3, WithRAGHybridSearch(), WithRAGQuery("explicit query")).
		User("user message")

	ctx := context.Background()
	b.resolveRAGContext(ctx)

	require.NotNil(t, capturedReq, "search should have been called")
	assert.Equal(t, "explicit query", capturedReq.TextQuery,
		"TextQuery should use the explicit query, not the user message")
}
