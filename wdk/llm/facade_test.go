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
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/cache/cache_domain"
	budget_memory "piko.sh/piko/internal/llm/llm_adapters/budget_store/memory"
	"piko.sh/piko/internal/llm/llm_adapters/memory_memory"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/wdk/llm"
	"piko.sh/piko/wdk/maths"
)

var (
	errNoBackend = errors.New("no vector backend")
)

func failingFactory(context.Context, string, *llm.VectorNamespaceConfig) (cache_domain.Cache[string, llm_dto.VectorDocument], error) {
	return nil, errNoBackend
}

func TestNewVectorStoreReportsFactoryFailures(t *testing.T) {
	store := llm.NewVectorStore(failingFactory, llm.WithVectorStoreMaxFilteredCandidates(64))
	t.Cleanup(func() { assert.NoError(t, store.Close(context.WithoutCancel(t.Context()))) })

	err := store.CreateNamespace(t.Context(), "docs", &llm.VectorNamespaceConfig{Dimension: 2})

	require.ErrorIs(t, err, errNoBackend)
}

func TestNewVectorStoreSearchesAnUnknownNamespace(t *testing.T) {
	store := llm.NewVectorStore(failingFactory)
	t.Cleanup(func() { assert.NoError(t, store.Close(context.WithoutCancel(t.Context()))) })

	response, err := store.Search(t.Context(), &llm.VectorSearchRequest{Namespace: "missing", Vector: []float32{1, 0}, TopK: 3})

	require.NoError(t, err)
	assert.Empty(t, response.Results)
	assert.Zero(t, response.TotalCount)
}

func TestValidateRequestRejectsAMissingModel(t *testing.T) {
	assert.ErrorIs(t, llm.ValidateRequest(&llm.CompletionRequest{}), llm.ErrEmptyModel)
}

func TestStripFrontmatterKeepsTheBody(t *testing.T) {
	document := llm.StripFrontmatter()(llm.Document{ID: "page", Content: "---\ntitle: Page\n---\nBody text"})

	assert.Equal(t, "Body text", document.Content)
}

func TestExtractFrontmatterCopiesSelectedKeys(t *testing.T) {
	transform := llm.ExtractFrontmatter(llm.WithFrontmatterKeys("title"), llm.WithFrontmatterPrefix("fm_"))

	document := transform(llm.Document{ID: "page", Content: "---\ntitle: Page\nauthor: Someone\n---\nBody"})

	assert.Equal(t, "Page", document.Metadata["fm_title"])
	assert.NotContains(t, document.Metadata, "fm_author")
}

func TestServiceConstructorsBuildUsableValues(t *testing.T) {
	service := llm.NewService("primary")

	assert.NotNil(t, llm.NewCompletionBuilder(service))
	assert.NotNil(t, llm.NewEmbeddingBuilder(service))
	assert.NotNil(t, llm.NewCostCalculator())
	assert.NotNil(t, llm.NewCostCalculatorWithPricing(llm.DefaultPricingTable))
}

func TestDefaultServiceNeedsAnInitialisedFramework(t *testing.T) {
	_, err := llm.GetDefaultService()
	require.Error(t, err)

	_, err = llm.NewCompletionBuilderFromDefault()
	require.Error(t, err)

	_, err = llm.NewEmbeddingBuilderFromDefault()
	require.Error(t, err)
}

func TestNewBudgetManagerRejectsRequestsOverThePerRequestLimit(t *testing.T) {
	manager := llm.NewBudgetManager(budget_memory.New(), llm.NewCostCalculator())
	manager.SetBudget("tenant", &llm.BudgetConfig{MaxCostPerRequest: maths.NewMoneyFromString("1.00", "USD")})

	require.True(t, manager.HasBudget("tenant"))
	assert.ErrorIs(t, manager.CheckBudget(t.Context(), "tenant", maths.NewMoneyFromString("5.00", "USD")), llm.ErrMaxCostExceeded)
	assert.NoError(t, manager.CheckBudget(t.Context(), "tenant", maths.NewMoneyFromString("0.50", "USD")))
}

func TestNewRateLimiterRejectsRequestsOverTheLimit(t *testing.T) {
	limiter := llm.NewRateLimiter(ratelimiter_adapters.NewInMemoryTokenBucketStore())
	limiter.SetLimits("tenant", 1, 0)

	require.NoError(t, limiter.Allow(t.Context(), "tenant"))
	assert.ErrorIs(t, limiter.Allow(t.Context(), "tenant"), llm.ErrRateLimited)
	assert.True(t, limiter.HasLimits("tenant"))
}

func TestNewCacheManagerStoresAndReturnsEntries(t *testing.T) {
	manager := llm.NewCacheManager(llm_domain.NewMockCacheStore(), time.Hour)
	entry := &llm.CacheEntry{
		Response:  &llm.CompletionResponse{Model: "model"},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Provider:  "mock",
		Model:     "model",
	}

	require.NoError(t, manager.Set(t.Context(), "key", entry))
	stored, err := manager.Get(t.Context(), "key")

	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "model", stored.Model)

	require.NoError(t, manager.Delete(t.Context(), "key"))
	missing, err := manager.Get(t.Context(), "key")
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestNewBufferMemoryKeepsOnlyTheNewestMessages(t *testing.T) {
	memory := llm.NewBufferMemory(memory_memory.New(), llm.WithBufferSize(2))

	for _, content := range []string{"first", "second", "third"} {
		require.NoError(t, memory.AddMessage(t.Context(), "chat", llm.NewUserMessage(content)))
	}
	messages, err := memory.GetMessages(t.Context(), "chat")

	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "second", messages[0].Content)
	assert.Equal(t, "third", messages[1].Content)
}

func TestNewWindowMemoryTrimsToTheTokenLimit(t *testing.T) {
	memory := llm.NewWindowMemory(memory_memory.New(), llm.WithTokenLimit(20))

	for _, content := range []string{"old " + strings.Repeat("a", 60), "new message"} {
		require.NoError(t, memory.AddMessage(t.Context(), "chat", llm.NewUserMessage(content)))
	}
	messages, err := memory.GetMessages(t.Context(), "chat")

	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "new message", messages[0].Content)
}

func TestNewSummaryMemoryReturnsBufferedMessages(t *testing.T) {
	memory := llm.NewSummaryMemory(memory_memory.New(), llm.NewService("primary"), llm.MemoryConfig{BufferSize: 4})

	require.NoError(t, memory.AddMessage(t.Context(), "chat", llm.NewUserMessage("hello")))
	messages, err := memory.GetMessages(t.Context(), "chat")

	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "hello", messages[0].Content)

	require.NoError(t, memory.Clear(t.Context(), "chat"))
}

func TestNewRecursiveCharacterSplitterSplitsLongDocuments(t *testing.T) {
	splitter, err := llm.NewRecursiveCharacterSplitter(10, 2)
	require.NoError(t, err)

	chunks := splitter.Split(llm.Document{ID: "doc", Content: "alpha beta gamma delta epsilon"})

	assert.Greater(t, len(chunks), 1)

	_, err = llm.NewRecursiveCharacterSplitter(5, 5)
	assert.Error(t, err)
}

func TestNewMarkdownSplitterSubSplitsOversizedSections(t *testing.T) {
	splitter, err := llm.NewMarkdownSplitter(24, 0, llm.WithMaxSplitLevel(3))
	require.NoError(t, err)

	chunks := splitter.Split(llm.Document{ID: "doc", Content: "First paragraph here.\n\nSecond paragraph here."})

	require.Len(t, chunks, 2)
	assert.Equal(t, "First paragraph here.", chunks[0].Content)
	assert.Equal(t, "Second paragraph here.", chunks[1].Content)

	_, err = llm.NewMarkdownSplitter(5, 5)
	assert.Error(t, err)
}

func TestPrependChunkContextAddsTitleAndHeading(t *testing.T) {
	document := llm.PrependChunkContext()(llm.Document{
		ID:       "chunk",
		Content:  "Body",
		Metadata: map[string]any{"doc_title": "Guide", "heading": "Intro"},
	})

	assert.Equal(t, "Guide > Intro\n\nBody", document.Content)
}

func TestNewRecursiveFSLoaderLoadsMatchingFiles(t *testing.T) {
	filesystem := fstest.MapFS{
		"docs/a.md":     {Data: []byte("alpha")},
		"docs/b.txt":    {Data: []byte("beta")},
		"docs/sub/c.md": {Data: []byte("gamma")},
	}

	documents, err := llm.NewRecursiveFSLoader(filesystem, "*.md").Load(t.Context())

	require.NoError(t, err)
	contents := make([]string, 0, len(documents))
	for _, document := range documents {
		contents = append(contents, document.Content)
	}
	assert.ElementsMatch(t, []string{"alpha", "gamma"}, contents)
}
