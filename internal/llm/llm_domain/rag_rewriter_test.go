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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestMergeSearchResults_Dedup(t *testing.T) {
	setA := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Score: 0.8, Content: "content 1"},
		{ID: "doc-2", Score: 0.7, Content: "content 2"},
	}
	setB := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Score: 0.9, Content: "content 1"},
		{ID: "doc-3", Score: 0.6, Content: "content 3"},
	}

	merged := mergeSearchResults([][]llm_dto.VectorSearchResult{setA, setB}, 10)

	require.Len(t, merged, 3)

	assert.Equal(t, "doc-1", merged[0].ID)
	assert.InDelta(t, 0.9, merged[0].Score, 0.001, "doc-1 should keep highest score")
	assert.Equal(t, "doc-2", merged[1].ID)
	assert.Equal(t, "doc-3", merged[2].ID)
}

func TestMergeSearchResults_TopK(t *testing.T) {
	setA := []llm_dto.VectorSearchResult{
		{ID: "doc-1", Score: 0.9},
		{ID: "doc-2", Score: 0.8},
		{ID: "doc-3", Score: 0.7},
	}
	setB := []llm_dto.VectorSearchResult{
		{ID: "doc-4", Score: 0.6},
		{ID: "doc-5", Score: 0.5},
	}

	merged := mergeSearchResults([][]llm_dto.VectorSearchResult{setA, setB}, 3)

	require.Len(t, merged, 3)
	assert.Equal(t, "doc-1", merged[0].ID)
	assert.Equal(t, "doc-2", merged[1].ID)
	assert.Equal(t, "doc-3", merged[2].ID)
}

func TestMergeSearchResults_Empty(t *testing.T) {
	merged := mergeSearchResults(nil, 5)
	assert.Empty(t, merged)

	merged = mergeSearchResults([][]llm_dto.VectorSearchResult{}, 5)
	assert.Empty(t, merged)
}

func TestWithRAGQueryRewriter(t *testing.T) {
	var config ragConfig

	called := false
	rewriter := func(_ context.Context, _ string) ([]string, error) {
		called = true
		return []string{"rewritten"}, nil
	}

	WithRAGQueryRewriter(rewriter)(&config)

	require.NotNil(t, config.rewriter)
	_, _ = config.rewriter(nil, "test")
	assert.True(t, called)
}

func TestParseRewriterResponse_Basic(t *testing.T) {
	content := "improved search query"
	queries := parseRewriterResponse(content, 1)

	require.Len(t, queries, 1)
	assert.Equal(t, "improved search query", queries[0])
}

func TestParseRewriterResponse_MultiLine(t *testing.T) {
	content := "query one\nquery two\nquery three"
	queries := parseRewriterResponse(content, 5)

	require.Len(t, queries, 3)
	assert.Equal(t, "query one", queries[0])
	assert.Equal(t, "query two", queries[1])
	assert.Equal(t, "query three", queries[2])
}

func TestParseRewriterResponse_StripNumbering(t *testing.T) {
	content := "1. first query\n2. second query\n3) third query"
	queries := parseRewriterResponse(content, 5)

	require.Len(t, queries, 3)
	assert.Equal(t, "first query", queries[0])
	assert.Equal(t, "second query", queries[1])
	assert.Equal(t, "third query", queries[2])
}

func TestParseRewriterResponse_EmptyLines(t *testing.T) {
	content := "\n  \nquery one\n\nquery two\n  \n"
	queries := parseRewriterResponse(content, 5)

	require.Len(t, queries, 2)
	assert.Equal(t, "query one", queries[0])
	assert.Equal(t, "query two", queries[1])
}

func TestParseRewriterResponse_MaxQueries(t *testing.T) {
	content := "one\ntwo\nthree\nfour\nfive"
	queries := parseRewriterResponse(content, 3)

	require.Len(t, queries, 3)
	assert.Equal(t, "one", queries[0])
	assert.Equal(t, "two", queries[1])
	assert.Equal(t, "three", queries[2])
}

func TestParseRewriterResponse_Empty(t *testing.T) {
	queries := parseRewriterResponse("", 5)
	assert.Empty(t, queries)

	queries = parseRewriterResponse("   \n  \n  ", 5)
	assert.Empty(t, queries)
}

func TestParseRewriterResponse_FiltersConversationalJunk(t *testing.T) {

	content := "Sure, here's a sample pk file:\n```\n.pk component example with template script and style"
	queries := parseRewriterResponse(content, 5)

	require.Len(t, queries, 1)
	assert.Equal(t, ".pk component example with template script and style", queries[0])
}

func TestIsRewriterJunk(t *testing.T) {
	tests := []struct {
		line string
		junk bool
	}{
		{line: "```go", junk: true},
		{line: "~~~", junk: true},
		{line: "Sure, here's the query:", junk: true},
		{line: "Here is your rewritten query:", junk: true},
		{line: "Of course! Here you go:", junk: true},
		{line: "Certainly, the rewritten query is:", junk: true},
		{line: "Let me rewrite that for you:", junk: true},
		{line: "The following query should work:", junk: true},
		{line: "Below are the queries:", junk: true},
		{line: "Note: this is just a suggestion", junk: true},
		{line: "Example: query here", junk: true},
		{line: "Output: something", junk: true},
		{line: "installation setup configuration", junk: false},
		{line: ".pk component template", junk: false},
		{line: "cache TTL configuration guide", junk: false},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assert.Equal(t, tt.junk, isRewriterJunk(tt.line))
		})
	}
}

func TestParseJSONRewriterResponse_Valid(t *testing.T) {
	content := `{"queries": ["installation guide", "setup configuration"]}`
	queries := parseJSONRewriterResponse(content, 5)

	require.Len(t, queries, 2)
	assert.Equal(t, "installation guide", queries[0])
	assert.Equal(t, "setup configuration", queries[1])
}

func TestParseJSONRewriterResponse_SingleQuery(t *testing.T) {
	content := `{"queries": ["cache TTL configuration"]}`
	queries := parseJSONRewriterResponse(content, 1)

	require.Len(t, queries, 1)
	assert.Equal(t, "cache TTL configuration", queries[0])
}

func TestParseJSONRewriterResponse_MaxQueries(t *testing.T) {
	content := `{"queries": ["one", "two", "three", "four"]}`
	queries := parseJSONRewriterResponse(content, 2)

	require.Len(t, queries, 2)
	assert.Equal(t, "one", queries[0])
	assert.Equal(t, "two", queries[1])
}

func TestParseJSONRewriterResponse_InvalidJSON(t *testing.T) {

	queries := parseJSONRewriterResponse("not json at all", 5)
	assert.Nil(t, queries)

	queries = parseJSONRewriterResponse("Sure, here's a query", 5)
	assert.Nil(t, queries)
}

func TestParseJSONRewriterResponse_EmptyQueries(t *testing.T) {

	queries := parseJSONRewriterResponse(`{"queries": []}`, 5)
	assert.Empty(t, queries)
}

func TestParseJSONRewriterResponse_WhitespaceInQueries(t *testing.T) {
	content := `{"queries": ["  trimmed  ", "", "  also trimmed  "]}`
	queries := parseJSONRewriterResponse(content, 5)

	require.Len(t, queries, 2)
	assert.Equal(t, "trimmed", queries[0])
	assert.Equal(t, "also trimmed", queries[1])
}

func TestParseJSONRewriterResponse_FallbackIntegration(t *testing.T) {

	content := "Sure, here's the query:\ninstallation setup guide"

	jsonQueries := parseJSONRewriterResponse(content, 5)
	assert.Nil(t, jsonQueries)

	textQueries := parseRewriterResponse(content, 5)
	require.Len(t, textQueries, 1)
	assert.Equal(t, "installation setup guide", textQueries[0])
}

func TestQueryRewriterOptions(t *testing.T) {
	config := queryRewriterConfig{
		maxQueries: defaultRewriterMaxQueries,
		maxTokens:  defaultRewriterMaxTokens,
	}

	WithRewriterModel("gpt-4o-mini")(&config)
	WithRewriterProvider("openai")(&config)
	WithRewriterPrompt("custom prompt")(&config)
	WithRewriterMaxQueries(5)(&config)
	WithRewriterMaxTokens(100)(&config)

	assert.Equal(t, "gpt-4o-mini", config.model)
	assert.Equal(t, "openai", config.provider)
	assert.Equal(t, "custom prompt", config.prompt)
	assert.Equal(t, 5, config.maxQueries)
	assert.Equal(t, 100, config.maxTokens)
}

func TestLLMQueryRewriter_SingleQuery(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.DefaultModelValue = "mock-model"
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{
			ID:      "rw-1",
			Model:   "mock-model",
			Created: time.Now().Unix(),
			Choices: []llm_dto.Choice{
				{
					Index: 0,
					Message: llm_dto.Message{
						Role:    llm_dto.RoleAssistant,
						Content: `{"queries":["improved query"]}`,
					},
					FinishReason: "stop",
				},
			},
		}, nil
	}

	service := NewService("mock")
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	rewriter := LLMQueryRewriter(service)
	queries, err := rewriter(context.Background(), "original query")

	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, "improved query", queries[0])
}

func TestLLMQueryRewriter_MultipleQueries(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.DefaultModelValue = "mock-model"
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{
			ID:      "rw-2",
			Model:   "mock-model",
			Created: time.Now().Unix(),
			Choices: []llm_dto.Choice{
				{
					Index: 0,
					Message: llm_dto.Message{
						Role:    llm_dto.RoleAssistant,
						Content: `{"queries":["alpha query","beta query","gamma query"]}`,
					},
					FinishReason: "stop",
				},
			},
		}, nil
	}

	service := NewService("mock")
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	rewriter := LLMQueryRewriter(service, WithRewriterMaxQueries(3))
	queries, err := rewriter(context.Background(), "broad search")

	require.NoError(t, err)
	require.Len(t, queries, 3)
	assert.Equal(t, "alpha query", queries[0])
	assert.Equal(t, "beta query", queries[1])
	assert.Equal(t, "gamma query", queries[2])
}

func TestLLMQueryRewriter_ProviderError(t *testing.T) {
	providerErr := errors.New("provider unavailable")

	provider := NewMockLLMProvider()
	provider.DefaultModelValue = "mock-model"
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, providerErr
	}

	service := NewService("mock")
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	rewriter := LLMQueryRewriter(service)
	queries, err := rewriter(context.Background(), "failing query")

	require.Error(t, err)
	assert.ErrorIs(t, err, providerErr)
	assert.Nil(t, queries)
}

func TestLLMQueryRewriter_EmptyResponse(t *testing.T) {
	provider := NewMockLLMProvider()
	provider.DefaultModelValue = "mock-model"
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{
			ID:      "rw-empty",
			Model:   "mock-model",
			Created: time.Now().Unix(),
			Choices: []llm_dto.Choice{
				{
					Index: 0,
					Message: llm_dto.Message{
						Role:    llm_dto.RoleAssistant,
						Content: "",
					},
					FinishReason: "stop",
				},
			},
		}, nil
	}

	service := NewService("mock")
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	rewriter := LLMQueryRewriter(service)
	queries, err := rewriter(context.Background(), "empty response query")

	require.NoError(t, err)
	assert.Nil(t, queries)
}
