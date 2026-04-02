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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/markdown/markdown_testparser"
)

var knowledgeBase = []llm_domain.Document{
	{
		ID:      "doc-piko-overview",
		Content: "Piko is a Go web framework that uses hexagonal architecture. It supports server-side rendering, caching, and background jobs. Piko applications are built using a modular WDK (Web Development Kit) system.",
		Metadata: map[string]any{
			"source": "docs/overview.md",
			"topic":  "architecture",
		},
	},
	{
		ID:      "doc-cache-system",
		Content: "Piko's cache system supports multiple providers: in-memory (Otter), Redis, and Valkey. Caches are created using NewCacheBuilder with typed keys and values. The cache supports TTL, search indexing, and vector fields for similarity search.",
		Metadata: map[string]any{
			"source": "docs/cache.md",
			"topic":  "caching",
		},
	},
	{
		ID:      "doc-llm-service",
		Content: "The LLM service provides completions, streaming, and embeddings. Providers like OpenAI, Anthropic, Gemini, Mistral, and Ollama can be registered. The service supports RAG via vector stores, cost tracking with budgets, and response caching.",
		Metadata: map[string]any{
			"source": "docs/llm.md",
			"topic":  "llm",
		},
	},
	{
		ID:      "doc-vector-search",
		Content: "Vector search in Piko uses HNSW (Hierarchical Navigable Small World) graphs for approximate nearest neighbour search. Embeddings are generated via the LLM service and stored in the vector store. Supported metrics are cosine similarity, euclidean distance, and dot product.",
		Metadata: map[string]any{
			"source": "docs/vector-search.md",
			"topic":  "search",
		},
	},
	{
		ID:      "doc-rate-limiter",
		Content: "Piko includes a rate limiter that supports fixed-window counters and token bucket algorithms. Rate limits are stored in the cache system. Configuration is per-route or global.",
		Metadata: map[string]any{
			"source": "docs/rate-limiter.md",
			"topic":  "middleware",
		},
	},
	{
		ID:      "doc-deployment",
		Content: "Piko applications can be deployed as standalone binaries, Docker containers, or WebAssembly modules. The framework supports SQLite for local development and PostgreSQL for production. Configuration can be loaded from environment variables, Vault, or cloud secret managers.",
		Metadata: map[string]any{
			"source": "docs/deployment.md",
			"topic":  "deployment",
		},
	},
}

func TestRAG_FullPipeline(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
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

func TestRAG_AddTextAndSearch(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddText(ctx, "notes", "note-1", "Go is a statically typed, compiled language designed at Google.")
	require.NoError(t, err)

	err = service.AddText(ctx, "notes", "note-2", "Rust is a systems programming language focused on safety and performance.")
	require.NoError(t, err)

	err = service.AddText(ctx, "notes", "note-3", "Python is a dynamically typed language popular for data science and scripting.")
	require.NoError(t, err)

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("compiled language from Google").
		Embed(ctx)
	require.NoError(t, err)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "notes",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      2,
	})
	require.NoError(t, err)
	require.NotEmpty(t, searchResp.Results)

	assert.Equal(t, "note-1", searchResp.Results[0].ID,
		"Go doc should be the best match for 'compiled language from Google'")
}

func TestRAG_MetadataFiltering(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "kb", knowledgeBase)
	require.NoError(t, err)

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How do I set up my application?").
		Embed(ctx)
	require.NoError(t, err)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "kb",
		Vector:          queryResp.Embeddings[0].Vector,
		TopK:            10,
		Filter:          map[string]any{"topic": "deployment"},
		IncludeMetadata: true,
	})
	require.NoError(t, err)

	require.Len(t, searchResp.Results, 1)
	assert.Equal(t, "doc-deployment", searchResp.Results[0].ID)
}

func TestRAG_MultipleNamespaces(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddText(ctx, "project-a", "a1", "Piko uses hexagonal architecture for clean separation of concerns.")
	require.NoError(t, err)

	err = service.AddText(ctx, "project-b", "b1", "The weather today is sunny with a high of 25 degrees.")
	require.NoError(t, err)

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("software architecture").
		Embed(ctx)
	require.NoError(t, err)

	searchA, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "project-a",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      5,
	})
	require.NoError(t, err)
	require.Len(t, searchA.Results, 1)
	assert.Equal(t, "a1", searchA.Results[0].ID)

	searchB, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "project-b",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      5,
	})
	require.NoError(t, err)
	require.Len(t, searchB.Results, 1)
	assert.Equal(t, "b1", searchB.Results[0].ID)
}

func TestRAG_StreamWithContext(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddText(ctx, "faq", "faq-1", "The default port for Piko development server is 8080. You can change it with the PORT environment variable.")
	require.NoError(t, err)

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("What port does the dev server use?").
		Embed(ctx)
	require.NoError(t, err)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace: "faq",
		Vector:    queryResp.Embeddings[0].Vector,
		TopK:      1,
	})
	require.NoError(t, err)
	require.NotEmpty(t, searchResp.Results)

	maxTokens := 500
	events, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		User("What port does the dev server use?").
		WithVectorContext(searchResp.Results).
		MaxTokens(maxTokens).
		Stream(ctx)
	require.NoError(t, err)

	var chunks []string
	var gotDone bool

	for event := range events {
		switch event.Type {
		case llm_dto.StreamEventChunk:
			if event.Chunk != nil && event.Chunk.Delta != nil && event.Chunk.Delta.Content != nil {
				chunks = append(chunks, *event.Chunk.Delta.Content)
			}
		case llm_dto.StreamEventDone:
			gotDone = true
		case llm_dto.StreamEventError:
			t.Fatalf("unexpected stream error: %v", event.Error)
		}
	}

	assert.True(t, gotDone)
	full := strings.Join(chunks, "")
	t.Logf("Streamed RAG answer: %s", full)
	assert.NotEmpty(t, full)
}

func TestRAG_MarkdownSplitterMetadata(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	docsPath := filepath.Join("..", "..", "..", "docs")

	err := service.NewIngest("md-meta").
		FromDirectory(docsPath, "get-started/*.md").
		Transform(llm_domain.ExtractFrontmatter()).
		Splitter(requireMarkdownSplitter(t, 500, 50)).
		PostSplitTransform(truncateChunks(500)).
		Do(ctx)
	require.NoError(t, err, "ingesting docs with ExtractFrontmatter + MarkdownSplitter")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("How do I get started with Piko?").
		Embed(ctx)
	require.NoError(t, err, "embedding query")

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "md-meta",
		Vector:          queryResp.Embeddings[0].Vector,
		TopK:            5,
		IncludeMetadata: true,
	})
	require.NoError(t, err, "searching vector store")
	require.NotEmpty(t, searchResp.Results, "expected search results")

	for i, r := range searchResp.Results {
		t.Logf("Result %d (id=%s, score=%.4f):", i+1, r.ID, r.Score)
		for k, v := range r.Metadata {
			t.Logf("  %s = %v", k, v)
		}
	}

	var hasHeading bool
	for _, r := range searchResp.Results {
		if _, ok := r.Metadata["heading"]; ok {
			hasHeading = true
			break
		}
	}
	assert.True(t, hasHeading, "at least one result should have heading metadata from MarkdownSplitter")

	var hasTitle bool
	for _, r := range searchResp.Results {
		if _, ok := r.Metadata["title"]; ok {
			hasTitle = true
			break
		}
	}
	assert.True(t, hasTitle, "at least one result should have title metadata from ExtractFrontmatter")

	var hasSource bool
	for _, r := range searchResp.Results {
		if _, ok := r.Metadata["source"]; ok {
			hasSource = true
			break
		}
	}
	assert.True(t, hasSource, "at least one result should have source metadata from file loader")
}

func TestRAG_ChainedTransforms(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	docsPath := filepath.Join("..", "..", "..", "docs")

	err := service.NewIngest("chained").
		FromDirectory(docsPath, "get-started/*.md").
		Transform(llm_domain.ExtractFrontmatter(
			llm_domain.WithFrontmatterKeys("title"),
			llm_domain.WithFrontmatterPrefix("doc_"),
		)).
		Transform(func(doc llm_domain.Document) llm_domain.Document {
			if doc.Metadata == nil {
				doc.Metadata = make(map[string]any)
			}
			doc.Metadata["pipeline"] = "chained"
			return doc
		}).
		Splitter(requireMarkdownSplitter(t, 500, 50)).
		PostSplitTransform(truncateChunks(500)).
		Do(ctx)
	require.NoError(t, err, "ingesting with chained transforms")

	queryResp, err := service.NewEmbedding().
		Model(globalEnv.embeddingModel).
		Input("getting started with Piko").
		Embed(ctx)
	require.NoError(t, err)

	searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
		Namespace:       "chained",
		Vector:          queryResp.Embeddings[0].Vector,
		TopK:            3,
		IncludeMetadata: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, searchResp.Results)

	for _, r := range searchResp.Results {
		assert.Equal(t, "chained", r.Metadata["pipeline"],
			"custom transform metadata should be present on result %s", r.ID)
	}

	var hasDocTitle bool
	for _, r := range searchResp.Results {
		if _, ok := r.Metadata["doc_title"]; ok {
			hasDocTitle = true
			break
		}
	}
	assert.True(t, hasDocTitle, "at least one result should have doc_title from prefixed ExtractFrontmatter")
}

func TestRAG_RealDocumentation(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 0, "cosine")
	service.SetVectorStore(vectorStore)

	docsPath := filepath.Join("..", "..", "..", "docs")

	err := service.NewIngest("piko-docs").
		FromDirectory(docsPath, "get-started/*.md").
		Transform(llm_domain.ExtractFrontmatter()).
		Splitter(requireMarkdownSplitter(t, 500, 50)).
		PostSplitTransform(truncateChunks(500)).
		Do(ctx)
	require.NoError(t, err, "ingesting real documentation")

	type searchQuery struct {
		query       string
		expectInTop string
	}

	queries := []searchQuery{
		{
			query:       "What are the core concepts of Piko?",
			expectInTop: "core-concepts",
		},
		{
			query:       "What is Piko and how does it work?",
			expectInTop: "introduction",
		},
		{
			query:       "How do I create my first page in Piko?",
			expectInTop: "first-page",
		},
	}

	for _, q := range queries {
		t.Run(q.query, func(t *testing.T) {
			queryResp, err := service.NewEmbedding().
				Model(globalEnv.embeddingModel).
				Input(q.query).
				Embed(ctx)
			require.NoError(t, err, "embedding query")
			require.Len(t, queryResp.Embeddings, 1)

			searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
				Namespace:       "piko-docs",
				Vector:          queryResp.Embeddings[0].Vector,
				TopK:            5,
				IncludeMetadata: true,
			})
			require.NoError(t, err, "searching vector store")
			require.NotEmpty(t, searchResp.Results, "expected search results")

			t.Logf("Query: %q", q.query)
			found := false
			for i, r := range searchResp.Results {
				t.Logf("  Result %d (score=%.4f, id=%s): %.100s...", i+1, r.Score, r.ID, r.Content)
				if strings.Contains(strings.ToLower(r.ID), q.expectInTop) {
					found = true
				}
			}
			assert.True(t, found,
				"expected a result containing %q in its ID among top 5 results", q.expectInTop)
		})
	}

	t.Run("full_rag_completion", func(t *testing.T) {
		queryResp, err := service.NewEmbedding().
			Model(globalEnv.embeddingModel).
			Input("How do I use the cache system in Piko?").
			Embed(ctx)
		require.NoError(t, err)

		searchResp, err := vectorStore.Search(ctx, &llm_dto.VectorSearchRequest{
			Namespace:       "piko-docs",
			Vector:          queryResp.Embeddings[0].Vector,
			TopK:            3,
			IncludeMetadata: true,
		})
		require.NoError(t, err)
		require.NotEmpty(t, searchResp.Results)

		maxTokens := 500
		response, err := service.NewCompletion().
			Model(globalEnv.completionModel).
			System("You are a helpful assistant that answers questions about the Piko web framework. Use ONLY the provided context documents. If the context doesn't contain the answer, say so.").
			User("How do I use the cache system in Piko?").
			WithVectorContext(searchResp.Results).
			MaxTokens(maxTokens).
			Do(ctx)
		require.NoError(t, err, "RAG completion with real docs")
		require.NotNil(t, response)
		require.NotEmpty(t, response.Choices)

		answer := response.Choices[0].Message.Content
		t.Logf("RAG answer from real docs: %s", answer)
		assert.NotEmpty(t, answer)
	})
}

func TestRAG_IngestContextCancellation(t *testing.T) {

	service, _ := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	docsPath := filepath.Join("..", "..", "..", "docs")

	ctx, cancel := context.WithDeadlineCause(t.Context(), time.Now().Add(-time.Second), fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	err := service.NewIngest("cancel-test").
		FromDirectory(docsPath, "**/*.md").
		Transform(llm_domain.ExtractFrontmatter()).
		Splitter(requireMarkdownSplitter(t, 500, 50)).
		Do(ctx)

	require.Error(t, err, "ingest with cancelled context should fail")
	assert.ErrorIs(t, err, context.DeadlineExceeded,
		"error should wrap context.DeadlineExceeded")
	t.Logf("Got expected error: %v", err)
}

func requireMarkdownSplitter(t *testing.T, chunkSize, overlap int) *llm_domain.MarkdownSplitter {
	t.Helper()
	s, err := llm_domain.NewMarkdownSplitter(chunkSize, overlap,
		llm_domain.WithSplitterMarkdownParser(markdown_testparser.NewParser()))
	require.NoError(t, err)
	return s
}
