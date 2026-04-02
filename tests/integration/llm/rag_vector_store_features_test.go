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

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestRAG_AutoDetectDimension(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 0, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "auto-dim", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	var embeddingDim int
	for _, doc := range knowledgeBase {
		stored, err := vectorStore.Get(ctx, "auto-dim", doc.ID)
		require.NoError(t, err, "retrieving stored document %s", doc.ID)
		require.NotNil(t, stored, "document %s should be in vector store", doc.ID)
		assert.NotEmpty(t, stored.Vector, "document %s should have an embedding vector", doc.ID)
		assert.Equal(t, doc.Content, stored.Content)

		if embeddingDim == 0 {
			embeddingDim = len(stored.Vector)
		} else {
			assert.Len(t, stored.Vector, embeddingDim,
				"all vectors should have the same auto-detected dimension")
		}
	}

	t.Logf("Auto-detected embedding dimension: %d", embeddingDim)
	assert.Greater(t, embeddingDim, 0, "embedding dimension should be positive")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How does caching work in Piko?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")
	require.Len(t, queryResp.Embeddings, 1)
	assert.Len(t, queryResp.Embeddings[0].Vector, embeddingDim,
		"query embedding dimension should match stored vectors")

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "auto-dim",
		Vector:          queryResp.Embeddings[0].Vector,
		TopK:            3,
		IncludeMetadata: true,
	})
	require.NoError(t, err, "searching vector store")
	require.NotEmpty(t, searchResp.Results, "expected search results")

	topIDs := make([]string, len(searchResp.Results))
	for i, r := range searchResp.Results {
		topIDs[i] = r.ID
		t.Logf("  Result %d (score=%.4f, id=%s): %.80s...", i+1, r.Score, r.ID, r.Content)
	}
	assert.Contains(t, topIDs, "doc-cache-system",
		"cache document should be in top results for a caching question")
}

func TestRAG_HybridSearch(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterHybridVectorStore(t)
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "hybrid-kb", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("cache providers TTL").
		Embed(ctx)
	require.NoError(t, err, "embedding query")
	require.Len(t, queryResp.Embeddings, 1)

	vectorOnly, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "hybrid-kb",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      6,
	})
	require.NoError(t, err, "vector-only search")
	require.NotEmpty(t, vectorOnly.Results, "vector-only should return results")

	t.Log("Vector-only results:")
	for i, r := range vectorOnly.Results {
		t.Logf("  %d. id=%s score=%.4f", i+1, r.ID, r.Score)
	}

	hybridResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "hybrid-kb",
		Vector:    queryResp.Embeddings[0].Vector,
		TextQuery: "Otter Redis TTL cache",
		TopK:      6,
	})
	require.NoError(t, err, "hybrid search")
	require.NotEmpty(t, hybridResp.Results, "hybrid should return results")

	t.Log("Hybrid results:")
	for i, r := range hybridResp.Results {
		t.Logf("  %d. id=%s score=%.4f", i+1, r.ID, r.Score)
	}

	hybridIDs := make([]string, len(hybridResp.Results))
	for i, r := range hybridResp.Results {
		hybridIDs[i] = r.ID
	}
	assert.Contains(t, hybridIDs, "doc-cache-system",
		"cache document should appear in hybrid results")
}

func TestRAG_HybridSearch_ViaPipeline(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterHybridVectorStore(t)
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "hybrid-pipeline", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer questions using ONLY the provided context. Be concise.").
		User("What cache providers does Piko support?").
		RAG("hybrid-pipeline", 3, llm_domain.WithRAGHybridSearch()).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "RAG completion with hybrid search")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)

	answer := response.Choices[0].Message.Content
	t.Logf("Hybrid RAG answer: %s", answer)
	assert.NotEmpty(t, answer)
	assert.NotEmpty(t, response.Sources, "should have RAG sources")

	found := false
	for _, src := range response.Sources {
		t.Logf("  Source: id=%s score=%.4f", src.ID, src.Score)
		if src.ID == "doc-cache-system" {
			found = true
		}
	}
	assert.True(t, found, "cache document should be among RAG sources")
}

func TestRAG_MinScoreFiltering(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 0, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "minscore", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How does caching work in Piko?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")

	allResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "minscore",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      10,
	})
	require.NoError(t, err, "search without threshold")
	require.NotEmpty(t, allResp.Results)

	t.Log("All results (no threshold):")
	for i, r := range allResp.Results {
		t.Logf("  %d. id=%s score=%.4f", i+1, r.ID, r.Score)
	}

	lowestScore := allResp.Results[len(allResp.Results)-1].Score
	highestScore := allResp.Results[0].Score

	midScore := (highestScore + lowestScore) / 2
	filteredResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "minscore",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      10,
		MinScore:  &midScore,
	})
	require.NoError(t, err, "search with mid threshold")

	t.Logf("Filtered results (minScore=%.4f):", midScore)
	for i, r := range filteredResp.Results {
		t.Logf("  %d. id=%s score=%.4f", i+1, r.ID, r.Score)
		assert.GreaterOrEqual(t, r.Score, midScore,
			"all filtered results should meet the minimum score threshold")
	}

	assert.Less(t, len(filteredResp.Results), len(allResp.Results),
		"filtered results should be a subset of all results")

	perfectResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "minscore",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      10,
		MinScore:  new(float32(0.999)),
	})
	require.NoError(t, err, "search with very high threshold")
	assert.Less(t, len(perfectResp.Results), len(allResp.Results),
		"near-perfect threshold should return fewer results than no threshold")
}

func TestRAG_DeleteNamespaceAndReIngest(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 0, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "del-test", knowledgeBase)
	require.NoError(t, err, "ingesting original documents")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How does caching work?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "del-test",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      3,
	})
	require.NoError(t, err, "search before deletion")
	require.NotEmpty(t, searchResp.Results, "should find results before deletion")

	err = vectorStore.DeleteNamespace(ctx, "del-test")
	require.NoError(t, err, "deleting namespace")

	emptyResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "del-test",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      3,
	})
	require.NoError(t, err, "search after deletion")
	assert.Empty(t, emptyResp.Results, "should find no results after namespace deletion")

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
			"old documents should not appear after re-ingestion, got %s", r.ID)
	}
}
