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

//go:build integration

package llm_integration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_adapters/vector_cache"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

const embeddingDimension = 384

func runRAGVectorStorePipeline(t *testing.T, vectorStore *vector_cache.Store) {
	t.Helper()

	service, ctx := createLLMService(t)
	service.SetVectorStore(vectorStore)

	err := vectorStore.CreateNamespace(ctx, "knowledge-base", &llm_domain.VectorNamespaceConfig{
		Dimension: embeddingDimension,
		Metric:    llm_dto.SimilarityCosine,
	})
	require.NoError(t, err, "creating namespace")

	err = service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	for _, doc := range knowledgeBase {
		stored, err := vectorStore.Get(ctx, "knowledge-base", doc.ID)
		require.NoError(t, err, "retrieving stored document %s", doc.ID)
		require.NotNil(t, stored, "document %s should be in vector store", doc.ID)
		assert.NotEmpty(t, stored.Vector, "document %s should have an embedding vector", doc.ID)
		assert.Equal(t, doc.Content, stored.Content)
	}

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How does caching work in Piko?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")
	require.Len(t, queryResp.Embeddings, 1)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "knowledge-base",
		Vector:          queryResp.Embeddings[0].Vector,
		TopK:            3,
		IncludeMetadata: true,
	})
	require.NoError(t, err, "searching vector store")
	require.NotEmpty(t, searchResp.Results, "expected search results")

	t.Logf("Query: 'How does caching work in Piko?'")
	for i, r := range searchResp.Results {
		t.Logf("  Result %d (score=%.4f, id=%s): %.80s...", i+1, r.Score, r.ID, r.Content)
	}

	topIDs := make([]string, len(searchResp.Results))
	for i, r := range searchResp.Results {
		topIDs[i] = r.ID
	}
	assert.Contains(t, topIDs, "doc-cache-system",
		"cache document should be in top results for a caching question")

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("You are a helpful assistant. Answer questions using ONLY the provided context documents. If the context doesn't contain the answer, say so.").
		User("How does caching work in Piko?").
		WithVectorContext(searchResp.Results).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "RAG completion")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)

	answer := response.Choices[0].Message.Content
	t.Logf("RAG answer: %s", answer)
	assert.NotEmpty(t, answer, "expected a non-empty answer")
}

func TestRAG_RedisStack_FullPipeline(t *testing.T) {

	vectorStore := createRedisVectorStore(t, embeddingDimension)
	runRAGVectorStorePipeline(t, vectorStore)
}

func TestRAG_Valkey_FullPipeline(t *testing.T) {

	vectorStore := createValkeyVectorStore(t, embeddingDimension)
	runRAGVectorStorePipeline(t, vectorStore)
}

func runDeleteNamespaceAndReIngest(t *testing.T, vectorStore *vector_cache.Store) {
	t.Helper()

	service, ctx := createLLMService(t)
	service.SetVectorStore(vectorStore)

	err := vectorStore.CreateNamespace(ctx, "del-test", &llm_domain.VectorNamespaceConfig{
		Dimension: embeddingDimension,
		Metric:    llm_dto.SimilarityCosine,
	})
	require.NoError(t, err, "creating namespace")

	err = service.AddDocuments(ctx, "del-test", knowledgeBase)
	require.NoError(t, err, "ingesting original documents")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How does caching work?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")
	require.Len(t, queryResp.Embeddings, 1)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "del-test",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      3,
	})
	require.NoError(t, err, "search before deletion")
	require.NotEmpty(t, searchResp.Results, "should find results before deletion")

	err = vectorStore.DeleteNamespace(ctx, "del-test")
	require.NoError(t, err, "deleting namespace")

	err = vectorStore.CreateNamespace(ctx, "del-test", &llm_domain.VectorNamespaceConfig{
		Dimension: embeddingDimension,
		Metric:    llm_dto.SimilarityCosine,
	})
	require.NoError(t, err, "re-creating namespace")

	newDocuments := []llm_domain.Document{
		{
			ID:      "new-doc-1",
			Content: "Bananas are a tropical fruit rich in potassium and fibre.",
		},
		{
			ID:      "new-doc-2",
			Content: "Quantum computing uses qubits for parallel computation.",
		},
	}

	err = service.AddDocuments(ctx, "del-test", newDocuments)
	require.NoError(t, err, "re-ingesting new documents")

	newQueryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("tropical fruit potassium").
		Embed(ctx)
	require.NoError(t, err, "embedding new query")

	newSearchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "del-test",
		Vector:    newQueryResp.Embeddings[0].Vector,
		TopK:      5,
	})
	require.NoError(t, err, "search after re-ingestion")
	require.NotEmpty(t, newSearchResp.Results, "should find results after re-ingestion")

	resultIDs := make([]string, len(newSearchResp.Results))
	for i, r := range newSearchResp.Results {
		resultIDs[i] = r.ID
	}
	assert.Contains(t, resultIDs, "new-doc-1", "new banana doc should appear in results")

	for _, r := range newSearchResp.Results {
		assert.False(t, strings.HasPrefix(r.ID, "doc-"),
			"old documents should not appear after deletion and re-ingestion, got %s", r.ID)
	}
}

func TestRAG_RedisStack_DeleteNamespace(t *testing.T) {

	vectorStore := createRedisVectorStore(t, embeddingDimension)
	runDeleteNamespaceAndReIngest(t, vectorStore)
}

func TestRAG_Valkey_DeleteNamespace(t *testing.T) {

	vectorStore := createValkeyVectorStore(t, embeddingDimension)
	runDeleteNamespaceAndReIngest(t, vectorStore)
}
