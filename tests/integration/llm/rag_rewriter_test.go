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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
)

func TestRAG_WithLLMRewriter_SingleQuery(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer questions using ONLY the provided context.").
		User("Tell me about Piko's caching?").
		RAG("knowledge-base", 3,
			llm_domain.WithRAGQueryRewriter(
				llm_domain.LLMQueryRewriter(service,
					llm_domain.WithRewriterModel(globalEnv.completionModel),
				),
			),
		).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "RAG completion with single-query rewriter")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)

	answer := response.Choices[0].Message.Content
	t.Logf("Rewritten single-query RAG answer: %s", answer)
	assert.NotEmpty(t, answer)
	assert.NotEmpty(t, response.Sources, "should have RAG sources")
}

func TestRAG_WithLLMRewriter_MultiQuery(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer questions using ONLY the provided context.").
		User("Tell me about Piko's caching?").
		RAG("knowledge-base", 3,
			llm_domain.WithRAGQueryRewriter(
				llm_domain.LLMQueryRewriter(service,
					llm_domain.WithRewriterModel(globalEnv.completionModel),
					llm_domain.WithRewriterMaxQueries(3),
				),
			),
		).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "RAG completion with multi-query rewriter")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)

	answer := response.Choices[0].Message.Content
	t.Logf("Rewritten multi-query RAG answer: %s", answer)
	assert.NotEmpty(t, answer)
	assert.NotEmpty(t, response.Sources, "should have RAG sources")
}

func TestRAG_WithCustomRewriter(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	rewriter := func(_ context.Context, _ string) ([]string, error) {
		return []string{
			"Piko cache system providers",
			"cache TTL configuration",
		}, nil
	}

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer questions using ONLY the provided context.").
		User("How does caching work?").
		RAG("knowledge-base", 3,
			llm_domain.WithRAGQueryRewriter(rewriter),
		).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "RAG completion with custom rewriter")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)
	assert.NotEmpty(t, response.Sources, "should have RAG sources")

	found := false
	for _, src := range response.Sources {
		if src.ID == "doc-cache-system" {
			found = true
			break
		}
	}
	assert.True(t, found, "cache document should be in sources with targeted rewrite")
}

func TestRAG_RewriterInDryRun(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	rewriter := func(_ context.Context, query string) ([]string, error) {
		return []string{
			query + " technical details",
			query + " code examples",
		}, nil
	}

	dump := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer using context.").
		User("How does caching work?").
		RAG("knowledge-base", 3,
			llm_domain.WithRAGQueryRewriter(rewriter),
		).
		MaxTokens(500).
		DryRun(ctx)

	assert.Equal(t, "How does caching work?", dump.OriginalQuery)
	require.Len(t, dump.RewrittenQueries, 2)
	assert.Equal(t, "How does caching work? technical details", dump.RewrittenQueries[0])
	assert.Equal(t, "How does caching work? code examples", dump.RewrittenQueries[1])
	assert.NotEmpty(t, dump.Sources, "DryRun should resolve RAG sources")

	t.Logf("DryRun dump:\n%s", dump.String())
	assert.Contains(t, dump.String(), "=== Query Rewriting ===")
}

func TestRAG_RewriterError_FallsBack(t *testing.T) {

	service, ctx := createLLMService(t)
	vectorStore := createOtterVectorStore(t, 3, "cosine")
	service.SetVectorStore(vectorStore)

	err := service.AddDocuments(ctx, "knowledge-base", knowledgeBase)
	require.NoError(t, err, "ingesting knowledge base documents")

	rewriter := func(_ context.Context, _ string) ([]string, error) {
		return nil, fmt.Errorf("simulated rewriter failure")
	}

	maxTokens := 500
	response, err := service.NewCompletion().
		Model(globalEnv.completionModel).
		System("Answer questions using ONLY the provided context.").
		User("How does caching work in Piko?").
		RAG("knowledge-base", 3,
			llm_domain.WithRAGQueryRewriter(rewriter),
		).
		MaxTokens(maxTokens).
		Do(ctx)
	require.NoError(t, err, "completion should succeed despite rewriter failure")
	require.NotNil(t, response)
	require.NotEmpty(t, response.Choices)

	answer := response.Choices[0].Message.Content
	t.Logf("Fallback RAG answer: %s", answer)
	assert.NotEmpty(t, answer)
	assert.NotEmpty(t, response.Sources, "should still have RAG sources from original query")
}
