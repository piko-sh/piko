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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/wdk/maths"
)

func TestMockLLMProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockLLMProvider
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "test-model"}

	response, err := m.Complete(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	eventChannel, err := m.Stream(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, eventChannel)

	for range eventChannel {
	}

	assert.False(t, m.SupportsStreaming())
	assert.False(t, m.SupportsStructuredOutput())
	assert.False(t, m.SupportsTools())
	assert.Empty(t, m.DefaultModel())

	models, err := m.ListModels(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, models)

	assert.NoError(t, m.Close(ctx))
}

func TestMockLLMProvider_Complete(t *testing.T) {
	t.Parallel()

	t.Run("nil CompleteFunc returns default response", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		ctx := context.Background()
		request := &llm_dto.CompletionRequest{Model: "gpt-4o"}

		response, err := m.Complete(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "mock-response-id", response.ID)
		assert.Equal(t, "gpt-4o", response.Model)
		require.Len(t, response.Choices, 1)
		assert.Equal(t, llm_dto.RoleAssistant, response.Choices[0].Message.Role)
		assert.Equal(t, "Mock response", response.Choices[0].Message.Content)
		require.NotNil(t, response.Usage)
		assert.Equal(t, 15, response.Usage.TotalTokens)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CompleteCallCount))
	})

	t.Run("delegates to CompleteFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		expected := &llm_dto.CompletionResponse{ID: "custom-id"}
		m.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return expected, nil
		}

		response, err := m.Complete(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		require.NoError(t, err)
		assert.Equal(t, expected, response)
	})

	t.Run("propagates error from CompleteFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		wantErr := errors.New("provider unavailable")
		m.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return nil, wantErr
		}

		response, err := m.Complete(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		assert.Nil(t, response)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockLLMProvider_Stream(t *testing.T) {
	t.Parallel()

	t.Run("nil StreamFunc returns default channel with done event", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		ctx := context.Background()
		request := &llm_dto.CompletionRequest{Model: "m"}

		eventChannel, err := m.Stream(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, eventChannel)

		var events []llm_dto.StreamEvent
		for ev := range eventChannel {
			events = append(events, ev)
		}
		require.Len(t, events, 1)
		assert.True(t, events[0].Done)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.StreamCallCount))
	})

	t.Run("delegates to StreamFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		customChannel := make(chan llm_dto.StreamEvent, 1)
		customChannel <- llm_dto.NewDoneEvent(nil)
		close(customChannel)

		m.StreamFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
			return customChannel, nil
		}

		eventChannel, err := m.Stream(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		require.NoError(t, err)
		require.NotNil(t, eventChannel)

		ev := <-eventChannel
		assert.True(t, ev.Done)
	})

	t.Run("propagates error from StreamFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		wantErr := errors.New("stream failure")
		m.StreamFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
			return nil, wantErr
		}

		eventChannel, err := m.Stream(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		assert.Nil(t, eventChannel)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockLLMProvider_SupportsStreaming(t *testing.T) {
	t.Parallel()

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.True(t, m.SupportsStreaming())
	})

	t.Run("returns configured value false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsStreamingValue = false
		assert.False(t, m.SupportsStreaming())
	})
}

func TestMockLLMProvider_SupportsStructuredOutput(t *testing.T) {
	t.Parallel()

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.True(t, m.SupportsStructuredOutput())
	})

	t.Run("returns configured value false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsStructuredValue = false
		assert.False(t, m.SupportsStructuredOutput())
	})
}

func TestMockLLMProvider_SupportsTools(t *testing.T) {
	t.Parallel()

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.True(t, m.SupportsTools())
	})

	t.Run("returns configured value false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsToolsValue = false
		assert.False(t, m.SupportsTools())
	})
}

func TestMockLLMProvider_SupportsPenalties(t *testing.T) {
	t.Parallel()

	t.Run("defaults to false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.False(t, m.SupportsPenalties())
	})

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsPenaltiesValue = true
		assert.True(t, m.SupportsPenalties())
	})
}

func TestMockLLMProvider_SupportsSeed(t *testing.T) {
	t.Parallel()

	t.Run("defaults to false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.False(t, m.SupportsSeed())
	})

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsSeedValue = true
		assert.True(t, m.SupportsSeed())
	})
}

func TestMockLLMProvider_SupportsParallelToolCalls(t *testing.T) {
	t.Parallel()

	t.Run("defaults to false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.False(t, m.SupportsParallelToolCalls())
	})

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsParallelToolCallsValue = true
		assert.True(t, m.SupportsParallelToolCalls())
	})
}

func TestMockLLMProvider_SupportsMessageName(t *testing.T) {
	t.Parallel()

	t.Run("defaults to false", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.False(t, m.SupportsMessageName())
	})

	t.Run("returns configured value true", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.SupportsMessageNameValue = true
		assert.True(t, m.SupportsMessageName())
	})
}

func TestMockLLMProvider_ListModels(t *testing.T) {
	t.Parallel()

	t.Run("nil ListModelsFunc returns default model list", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		models, err := m.ListModels(context.Background())

		require.NoError(t, err)
		require.Len(t, models, 1)
		assert.Equal(t, "mock-model", models[0].ID)
	})

	t.Run("delegates to ListModelsFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		expected := []llm_dto.ModelInfo{{ID: "custom-model", Name: "Custom"}}
		m.ListModelsFunc = func(_ context.Context) ([]llm_dto.ModelInfo, error) {
			return expected, nil
		}

		models, err := m.ListModels(context.Background())

		require.NoError(t, err)
		assert.Equal(t, expected, models)
	})

	t.Run("propagates error from ListModelsFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		wantErr := errors.New("list models failure")
		m.ListModelsFunc = func(_ context.Context) ([]llm_dto.ModelInfo, error) {
			return nil, wantErr
		}

		models, err := m.ListModels(context.Background())

		assert.Nil(t, models)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockLLMProvider_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns nil", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.NoError(t, m.Close(context.Background()))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		called := false
		m.CloseFunc = func(_ context.Context) error {
			called = true
			return nil
		}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		wantErr := errors.New("close failure")
		m.CloseFunc = func(_ context.Context) error {
			return wantErr
		}

		err := m.Close(context.Background())

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockLLMProvider_DefaultModel(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string when not configured", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		assert.Empty(t, m.DefaultModel())
	})

	t.Run("returns configured value", func(t *testing.T) {
		t.Parallel()

		m := NewMockLLMProvider()
		m.DefaultModelValue = "gpt-4o"
		assert.Equal(t, "gpt-4o", m.DefaultModel())
	})
}

func TestMockLLMProvider_SetResponse(t *testing.T) {
	t.Parallel()

	m := NewMockLLMProvider()
	expected := &llm_dto.CompletionResponse{ID: "set-response-id"}
	m.SetResponse(expected)

	response, err := m.Complete(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

	require.NoError(t, err)
	assert.Equal(t, expected, response)
}

func TestMockLLMProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockLLMProvider()
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "m"}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			_, _ = m.Complete(ctx, request)
			eventChannel, _ := m.Stream(ctx, request)
			if eventChannel != nil {
				for range eventChannel {
				}
			}
			_ = m.SupportsStreaming()
			_ = m.SupportsStructuredOutput()
			_ = m.SupportsTools()
			_ = m.DefaultModel()
			_, _ = m.ListModels(ctx)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CompleteCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.StreamCallCount))
}

func TestMockEmbeddingProvider_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockEmbeddingProvider
	ctx := context.Background()
	request := &llm_dto.EmbeddingRequest{Model: "test", Input: []string{"hello"}}

	response, err := m.Embed(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	models, err := m.ListEmbeddingModels(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, models)

	assert.Equal(t, 0, m.EmbeddingDimensions())
	assert.NoError(t, m.Close(ctx))
}

func TestMockEmbeddingProvider_Embed(t *testing.T) {
	t.Parallel()

	t.Run("nil EmbedFunc returns default response", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3",
			Input: []string{"hello", "world"},
		}

		response, err := m.Embed(context.Background(), request)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "text-embedding-3", response.Model)
		require.Len(t, response.Embeddings, 2)
		assert.Equal(t, []float32{0.1, 0.2, 0.3}, response.Embeddings[0].Vector)
		assert.Equal(t, 0, response.Embeddings[0].Index)
		assert.Equal(t, 1, response.Embeddings[1].Index)
		require.NotNil(t, response.Usage)
		assert.Equal(t, 10, response.Usage.PromptTokens)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.EmbedCallCount))
	})

	t.Run("delegates to EmbedFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		expected := &llm_dto.EmbeddingResponse{Model: "custom"}
		m.EmbedFunc = func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
			return expected, nil
		}

		response, err := m.Embed(context.Background(), &llm_dto.EmbeddingRequest{Model: "m", Input: []string{"x"}})

		require.NoError(t, err)
		assert.Equal(t, expected, response)
	})

	t.Run("propagates error from EmbedFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		wantErr := errors.New("embed failure")
		m.EmbedFunc = func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
			return nil, wantErr
		}

		response, err := m.Embed(context.Background(), &llm_dto.EmbeddingRequest{Model: "m", Input: []string{"x"}})

		assert.Nil(t, response)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockEmbeddingProvider_SetEmbedFunc(t *testing.T) {
	t.Parallel()

	m := NewMockEmbeddingProvider()
	expected := &llm_dto.EmbeddingResponse{Model: "via-setter"}
	m.SetEmbedFunc(func(_ context.Context, _ *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		return expected, nil
	})

	response, err := m.Embed(context.Background(), &llm_dto.EmbeddingRequest{Model: "m", Input: []string{"x"}})

	require.NoError(t, err)
	assert.Equal(t, expected, response)
}

func TestMockEmbeddingProvider_ListEmbeddingModels(t *testing.T) {
	t.Parallel()

	t.Run("nil ListModelsFunc returns default model list", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		models, err := m.ListEmbeddingModels(context.Background())

		require.NoError(t, err)
		require.Len(t, models, 1)
		assert.Equal(t, "mock-embedding-model", models[0].ID)
	})

	t.Run("delegates to ListModelsFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		expected := []llm_dto.ModelInfo{{ID: "custom-embed-model"}}
		m.ListModelsFunc = func(_ context.Context) ([]llm_dto.ModelInfo, error) {
			return expected, nil
		}

		models, err := m.ListEmbeddingModels(context.Background())

		require.NoError(t, err)
		assert.Equal(t, expected, models)
	})

	t.Run("propagates error from ListModelsFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		wantErr := errors.New("list models failure")
		m.ListModelsFunc = func(_ context.Context) ([]llm_dto.ModelInfo, error) {
			return nil, wantErr
		}

		models, err := m.ListEmbeddingModels(context.Background())

		assert.Nil(t, models)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockEmbeddingProvider_EmbeddingDimensions(t *testing.T) {
	t.Parallel()

	t.Run("nil EmbeddingDimensionsFunc returns zero", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		assert.Equal(t, 0, m.EmbeddingDimensions())
	})

	t.Run("delegates to EmbeddingDimensionsFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		m.EmbeddingDimensionsFunc = func() int { return 1536 }

		assert.Equal(t, 1536, m.EmbeddingDimensions())
	})
}

func TestMockEmbeddingProvider_Close(t *testing.T) {
	t.Parallel()

	t.Run("nil CloseFunc returns nil", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		assert.NoError(t, m.Close(context.Background()))
	})

	t.Run("delegates to CloseFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		called := false
		m.CloseFunc = func(_ context.Context) error {
			called = true
			return nil
		}

		err := m.Close(context.Background())

		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("propagates error from CloseFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockEmbeddingProvider()
		wantErr := errors.New("close failure")
		m.CloseFunc = func(_ context.Context) error {
			return wantErr
		}

		err := m.Close(context.Background())

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockEmbeddingProvider_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockEmbeddingProvider()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()

			request := &llm_dto.EmbeddingRequest{Model: "m", Input: []string{"hello"}}
			_, _ = m.Embed(ctx, request)
			_, _ = m.ListEmbeddingModels(ctx)
			_ = m.EmbeddingDimensions()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.EmbedCallCount))
}

func TestMockCacheStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := NewMockCacheStore()
	ctx := context.Background()

	entry, err := m.Get(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, entry)

	assert.NoError(t, m.Set(ctx, "k", &llm_dto.CacheEntry{}))
	assert.NoError(t, m.Delete(ctx, "k"))
	assert.NoError(t, m.Clear(ctx))

	stats, err := m.GetStats(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)
}

func TestMockCacheStore_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil GetFunc returns nil for missing key", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()

		entry, err := m.Get(context.Background(), "missing")

		assert.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("nil GetFunc returns stored entry", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		ctx := context.Background()
		expected := &llm_dto.CacheEntry{
			Response: &llm_dto.CompletionResponse{ID: "cached"},
		}

		require.NoError(t, m.Set(ctx, "key1", expected))

		entry, err := m.Get(ctx, "key1")

		require.NoError(t, err)
		assert.Equal(t, expected, entry)
	})

	t.Run("delegates to GetFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		expected := &llm_dto.CacheEntry{
			Response: &llm_dto.CompletionResponse{ID: "from-func"},
		}
		m.GetFunc = func(_ context.Context, _ string) (*llm_dto.CacheEntry, error) {
			return expected, nil
		}

		entry, err := m.Get(context.Background(), "any")

		require.NoError(t, err)
		assert.Equal(t, expected, entry)
	})

	t.Run("propagates error from GetFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		wantErr := errors.New("cache get failure")
		m.GetFunc = func(_ context.Context, _ string) (*llm_dto.CacheEntry, error) {
			return nil, wantErr
		}

		entry, err := m.Get(context.Background(), "any")

		assert.Nil(t, entry)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCacheStore_Set(t *testing.T) {
	t.Parallel()

	t.Run("nil SetFunc stores entry in map", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		ctx := context.Background()
		entry := &llm_dto.CacheEntry{
			Response: &llm_dto.CompletionResponse{ID: "stored"},
		}

		err := m.Set(ctx, "k", entry)
		require.NoError(t, err)

		got, err := m.Get(ctx, "k")
		require.NoError(t, err)
		assert.Equal(t, entry, got)
	})

	t.Run("delegates to SetFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		var capturedKey string
		m.SetFunc = func(_ context.Context, key string, _ *llm_dto.CacheEntry) error {
			capturedKey = key
			return nil
		}

		err := m.Set(context.Background(), "mykey", &llm_dto.CacheEntry{})

		assert.NoError(t, err)
		assert.Equal(t, "mykey", capturedKey)
	})

	t.Run("propagates error from SetFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		wantErr := errors.New("cache set failure")
		m.SetFunc = func(_ context.Context, _ string, _ *llm_dto.CacheEntry) error {
			return wantErr
		}

		err := m.Set(context.Background(), "k", &llm_dto.CacheEntry{})

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockCacheStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes existing entry", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		ctx := context.Background()

		require.NoError(t, m.Set(ctx, "k", &llm_dto.CacheEntry{}))

		err := m.Delete(ctx, "k")
		require.NoError(t, err)

		got, err := m.Get(ctx, "k")
		assert.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("deleting nonexistent key is safe", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		assert.NoError(t, m.Delete(context.Background(), "nonexistent"))
	})
}

func TestMockCacheStore_Clear(t *testing.T) {
	t.Parallel()

	m := NewMockCacheStore()
	ctx := context.Background()

	require.NoError(t, m.Set(ctx, "a", &llm_dto.CacheEntry{}))
	require.NoError(t, m.Set(ctx, "b", &llm_dto.CacheEntry{}))

	err := m.Clear(ctx)
	require.NoError(t, err)

	entryA, err := m.Get(ctx, "a")
	assert.NoError(t, err)
	assert.Nil(t, entryA)

	entryB, err := m.Get(ctx, "b")
	assert.NoError(t, err)
	assert.Nil(t, entryB)
}

func TestMockCacheStore_GetStats(t *testing.T) {
	t.Parallel()

	t.Run("returns zeroes on empty cache", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()

		stats, err := m.GetStats(context.Background())

		require.NoError(t, err)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, int64(0), stats.Size)
	})

	t.Run("tracks hits and misses", func(t *testing.T) {
		t.Parallel()

		m := NewMockCacheStore()
		ctx := context.Background()

		require.NoError(t, m.Set(ctx, "k", &llm_dto.CacheEntry{}))

		_, _ = m.Get(ctx, "k")

		_, _ = m.Get(ctx, "missing1")
		_, _ = m.Get(ctx, "missing2")

		stats, err := m.GetStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(2), stats.Misses)
		assert.Equal(t, int64(1), stats.Size)
	})
}

func TestMockCacheStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockCacheStore()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()

			key := fmt.Sprintf("key-%d", index)
			_ = m.Set(ctx, key, &llm_dto.CacheEntry{})
			_, _ = m.Get(ctx, key)
			_ = m.Delete(ctx, key)
			_, _ = m.GetStats(ctx)
		}(i)
	}

	wg.Wait()

	stats, err := m.GetStats(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.Hits, int64(0))
}

func TestMockBudgetStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := NewMockBudgetStore()
	ctx := context.Background()

	status, err := m.GetStatus(ctx, "global")
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "global", status.Scope)

	assert.NoError(t, m.IncrementRequests(ctx, "global", 1))
	assert.NoError(t, m.IncrementTokens(ctx, "global", 100))
	assert.NoError(t, m.Reset(ctx, "global"))
}

func TestMockBudgetStore_Record(t *testing.T) {
	t.Parallel()

	t.Run("nil RecordFunc accumulates costs", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()
		cost := &llm_dto.CostEstimate{
			TotalCost: maths.ZeroMoney(llm_dto.CostCurrency),
			Timestamp: time.Now(),
		}

		err := m.Record(ctx, "scope1", cost)

		require.NoError(t, err)

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)
		assert.Equal(t, "scope1", status.Scope)
	})

	t.Run("delegates to RecordFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		var capturedScope string
		m.RecordFunc = func(_ context.Context, scope string, _ *llm_dto.CostEstimate) error {
			capturedScope = scope
			return nil
		}

		cost := &llm_dto.CostEstimate{
			TotalCost: maths.ZeroMoney(llm_dto.CostCurrency),
		}
		err := m.Record(context.Background(), "test-scope", cost)

		assert.NoError(t, err)
		assert.Equal(t, "test-scope", capturedScope)
	})

	t.Run("propagates error from RecordFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		wantErr := errors.New("record failure")
		m.RecordFunc = func(_ context.Context, _ string, _ *llm_dto.CostEstimate) error {
			return wantErr
		}

		cost := &llm_dto.CostEstimate{
			TotalCost: maths.ZeroMoney(llm_dto.CostCurrency),
		}
		err := m.Record(context.Background(), "scope", cost)

		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockBudgetStore_GetStatus(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStatusFunc returns zero status for unknown scope", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()

		status, err := m.GetStatus(context.Background(), "unknown")

		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "unknown", status.Scope)
	})

	t.Run("nil GetStatusFunc returns recorded status", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()
		cost := &llm_dto.CostEstimate{
			TotalCost: maths.ZeroMoney(llm_dto.CostCurrency),
		}

		require.NoError(t, m.Record(ctx, "scope1", cost))

		status, err := m.GetStatus(ctx, "scope1")

		require.NoError(t, err)
		assert.Equal(t, "scope1", status.Scope)
	})

	t.Run("delegates to GetStatusFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		expected := &llm_dto.BudgetStatus{Scope: "custom"}
		m.GetStatusFunc = func(_ context.Context, _ string) (*llm_dto.BudgetStatus, error) {
			return expected, nil
		}

		status, err := m.GetStatus(context.Background(), "any")

		require.NoError(t, err)
		assert.Equal(t, expected, status)
	})

	t.Run("propagates error from GetStatusFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		wantErr := errors.New("get status failure")
		m.GetStatusFunc = func(_ context.Context, _ string) (*llm_dto.BudgetStatus, error) {
			return nil, wantErr
		}

		status, err := m.GetStatus(context.Background(), "scope")

		assert.Nil(t, status)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockBudgetStore_IncrementRequests(t *testing.T) {
	t.Parallel()

	t.Run("increments request count for new scope", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()

		err := m.IncrementRequests(ctx, "scope1", 5)
		require.NoError(t, err)

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)
		assert.Equal(t, int64(5), status.RequestCount)
	})

	t.Run("accumulates request counts", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()

		require.NoError(t, m.IncrementRequests(ctx, "scope1", 3))
		require.NoError(t, m.IncrementRequests(ctx, "scope1", 7))

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)
		assert.Equal(t, int64(10), status.RequestCount)
	})
}

func TestMockBudgetStore_IncrementTokens(t *testing.T) {
	t.Parallel()

	t.Run("increments token count for new scope", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()

		err := m.IncrementTokens(ctx, "scope1", 100)
		require.NoError(t, err)

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)
		assert.Equal(t, int64(100), status.TokenCount)
	})

	t.Run("accumulates token counts", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()

		require.NoError(t, m.IncrementTokens(ctx, "scope1", 50))
		require.NoError(t, m.IncrementTokens(ctx, "scope1", 150))

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)
		assert.Equal(t, int64(200), status.TokenCount)
	})
}

func TestMockBudgetStore_Reset(t *testing.T) {
	t.Parallel()

	t.Run("removes scope data", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		ctx := context.Background()

		require.NoError(t, m.IncrementRequests(ctx, "scope1", 10))
		require.NoError(t, m.Reset(ctx, "scope1"))

		status, err := m.GetStatus(ctx, "scope1")
		require.NoError(t, err)

		assert.Equal(t, int64(0), status.RequestCount)
	})

	t.Run("resetting nonexistent scope is safe", func(t *testing.T) {
		t.Parallel()

		m := NewMockBudgetStore()
		assert.NoError(t, m.Reset(context.Background(), "nonexistent"))
	})
}

func TestMockBudgetStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockBudgetStore()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()

			scope := fmt.Sprintf("scope-%d", index%5)
			cost := &llm_dto.CostEstimate{
				TotalCost: maths.ZeroMoney(llm_dto.CostCurrency),
			}
			_ = m.Record(ctx, scope, cost)
			_, _ = m.GetStatus(ctx, scope)
			_ = m.IncrementRequests(ctx, scope, 1)
			_ = m.IncrementTokens(ctx, scope, 10)
		}(i)
	}

	wg.Wait()

	for i := range 5 {
		scope := fmt.Sprintf("scope-%d", i)
		status, err := m.GetStatus(ctx, scope)
		require.NoError(t, err)
		assert.NotNil(t, status)
	}
}

func TestMockMemoryStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := NewMockMemoryStore()
	ctx := context.Background()

	_, err := m.Load(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrConversationNotFound)

	ids, err := m.List(ctx, "")
	assert.NoError(t, err)
	assert.Empty(t, ids)
}

func TestMockMemoryStore_Load(t *testing.T) {
	t.Parallel()

	t.Run("returns ErrConversationNotFound for missing ID", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()

		state, err := m.Load(context.Background(), "missing")

		assert.Nil(t, state)
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})

	t.Run("returns saved state", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		ctx := context.Background()
		expected := &llm_dto.ConversationState{ID: "conv-1"}

		require.NoError(t, m.Save(ctx, expected))

		state, err := m.Load(ctx, "conv-1")

		require.NoError(t, err)
		assert.Equal(t, expected, state)
	})
}

func TestMockMemoryStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("stores conversation state", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		ctx := context.Background()
		state := &llm_dto.ConversationState{ID: "conv-1"}

		err := m.Save(ctx, state)

		require.NoError(t, err)

		loaded, err := m.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, state, loaded)
	})

	t.Run("overwrites existing state", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		ctx := context.Background()
		original := &llm_dto.ConversationState{ID: "conv-1"}
		updated := &llm_dto.ConversationState{
			ID:        "conv-1",
			UpdatedAt: time.Now(),
		}

		require.NoError(t, m.Save(ctx, original))
		require.NoError(t, m.Save(ctx, updated))

		loaded, err := m.Load(ctx, "conv-1")
		require.NoError(t, err)
		assert.Equal(t, updated, loaded)
	})
}

func TestMockMemoryStore_Delete(t *testing.T) {
	t.Parallel()

	t.Run("removes existing state", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		ctx := context.Background()
		state := &llm_dto.ConversationState{ID: "conv-1"}

		require.NoError(t, m.Save(ctx, state))
		require.NoError(t, m.Delete(ctx, "conv-1"))

		_, err := m.Load(ctx, "conv-1")
		assert.ErrorIs(t, err, ErrConversationNotFound)
	})

	t.Run("deleting nonexistent ID is safe", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		assert.NoError(t, m.Delete(context.Background(), "nonexistent"))
	})
}

func TestMockMemoryStore_List(t *testing.T) {
	t.Parallel()

	t.Run("returns empty list when store is empty", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()

		ids, err := m.List(context.Background(), "")

		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("returns all stored conversation IDs", func(t *testing.T) {
		t.Parallel()

		m := NewMockMemoryStore()
		ctx := context.Background()

		require.NoError(t, m.Save(ctx, &llm_dto.ConversationState{ID: "conv-1"}))
		require.NoError(t, m.Save(ctx, &llm_dto.ConversationState{ID: "conv-2"}))

		ids, err := m.List(ctx, "")

		require.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.ElementsMatch(t, []string{"conv-1", "conv-2"}, ids)
	})
}

func TestMockMemoryStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockMemoryStore()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()

			id := fmt.Sprintf("conv-%d", index)
			state := &llm_dto.ConversationState{ID: id}

			_ = m.Save(ctx, state)
			_, _ = m.Load(ctx, id)
			_, _ = m.List(ctx, "")
			_ = m.Delete(ctx, id)
		}(i)
	}

	wg.Wait()

	ids, err := m.List(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestMockSummariser_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m MockSummariser
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "test-model"}

	response, err := m.Complete(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestMockSummariser_Complete(t *testing.T) {
	t.Parallel()

	t.Run("nil CompleteFunc returns default summary", func(t *testing.T) {
		t.Parallel()

		m := NewMockSummariser()
		request := &llm_dto.CompletionRequest{Model: "gpt-4o"}

		response, err := m.Complete(context.Background(), request)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "mock-summary-id", response.ID)
		assert.Equal(t, "gpt-4o", response.Model)
		require.Len(t, response.Choices, 1)
		assert.Contains(t, response.Choices[0].Message.Content, "mock summary")
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CompleteCallCount))
	})

	t.Run("delegates to CompleteFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockSummariser()
		expected := &llm_dto.CompletionResponse{ID: "custom-summary"}
		m.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return expected, nil
		}

		response, err := m.Complete(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		require.NoError(t, err)
		assert.Equal(t, expected, response)
	})

	t.Run("propagates error from CompleteFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockSummariser()
		wantErr := errors.New("summariser failure")
		m.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return nil, wantErr
		}

		response, err := m.Complete(context.Background(), &llm_dto.CompletionRequest{Model: "m"})

		assert.Nil(t, response)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockSummariser_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockSummariser()
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "m"}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.Complete(ctx, request)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CompleteCallCount))
}

func TestMockRateLimiterStore_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := NewMockRateLimiterStore()
	ctx := context.Background()
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	allowed, err := m.TryTake(ctx, "key", 1, config)
	assert.NoError(t, err)
	assert.True(t, allowed)

	dur, err := m.WaitDuration(ctx, "key", 1, config)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dur, time.Duration(0))

	assert.NoError(t, m.DeleteBucket(ctx, "key"))
}

func TestMockRateLimiterStore_TryTake(t *testing.T) {
	t.Parallel()

	t.Run("nil TryTakeFunc allows take when bucket has tokens", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 100.0, Burst: 100}

		allowed, err := m.TryTake(context.Background(), "bucket1", 1, config)

		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("nil TryTakeFunc denies take when bucket is exhausted", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

		allowed, err := m.TryTake(context.Background(), "bucket1", 1, config)
		require.NoError(t, err)
		require.True(t, allowed)

		allowed, err = m.TryTake(context.Background(), "bucket1", 1, config)
		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("delegates to TryTakeFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		m.TryTakeFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return false, nil
		}

		allowed, err := m.TryTake(context.Background(), "key", 1, &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 10})

		require.NoError(t, err)
		assert.False(t, allowed)
	})

	t.Run("propagates error from TryTakeFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		wantErr := errors.New("rate limiter failure")
		m.TryTakeFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (bool, error) {
			return false, wantErr
		}

		allowed, err := m.TryTake(context.Background(), "key", 1, &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 10})

		assert.False(t, allowed)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRateLimiterStore_WaitDuration(t *testing.T) {
	t.Parallel()

	t.Run("nil WaitDurationFunc returns zero for unknown bucket", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

		dur, err := m.WaitDuration(context.Background(), "unknown", 1, config)

		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), dur)
	})

	t.Run("nil WaitDurationFunc returns zero when tokens are available", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 100.0, Burst: 100}

		_, _ = m.TryTake(context.Background(), "bucket1", 1, config)

		dur, err := m.WaitDuration(context.Background(), "bucket1", 1, config)

		require.NoError(t, err)
		assert.Equal(t, time.Duration(0), dur)
	})

	t.Run("nil WaitDurationFunc returns positive duration when depleted", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

		_, _ = m.TryTake(context.Background(), "bucket1", 1, config)

		dur, err := m.WaitDuration(context.Background(), "bucket1", 1, config)

		require.NoError(t, err)
		assert.Greater(t, dur, time.Duration(0))
	})

	t.Run("nil WaitDurationFunc returns hour when refill rate is zero", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 0, Burst: 1}

		_, _ = m.TryTake(context.Background(), "zero-rate", 1, config)

		dur, err := m.WaitDuration(context.Background(), "zero-rate", 1, config)

		require.NoError(t, err)
		assert.Equal(t, time.Hour, dur)
	})

	t.Run("delegates to WaitDurationFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		m.WaitDurationFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
			return 42 * time.Second, nil
		}

		dur, err := m.WaitDuration(context.Background(), "key", 1, &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 10})

		require.NoError(t, err)
		assert.Equal(t, 42*time.Second, dur)
	})

	t.Run("propagates error from WaitDurationFunc", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		wantErr := errors.New("wait duration failure")
		m.WaitDurationFunc = func(_ context.Context, _ string, _ float64, _ *ratelimiter_dto.TokenBucketConfig) (time.Duration, error) {
			return 0, wantErr
		}

		dur, err := m.WaitDuration(context.Background(), "key", 1, &ratelimiter_dto.TokenBucketConfig{Rate: 10, Burst: 10})

		assert.Equal(t, time.Duration(0), dur)
		assert.ErrorIs(t, err, wantErr)
	})
}

func TestMockRateLimiterStore_DeleteBucket(t *testing.T) {
	t.Parallel()

	t.Run("removes existing bucket", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		ctx := context.Background()
		config := &ratelimiter_dto.TokenBucketConfig{Rate: 1.0, Burst: 1}

		_, _ = m.TryTake(ctx, "bucket1", 1, config)

		err := m.DeleteBucket(ctx, "bucket1")
		require.NoError(t, err)

		allowed, err := m.TryTake(ctx, "bucket1", 1, config)
		require.NoError(t, err)
		assert.True(t, allowed)
	})

	t.Run("deleting nonexistent bucket is safe", func(t *testing.T) {
		t.Parallel()

		m := NewMockRateLimiterStore()
		assert.NoError(t, m.DeleteBucket(context.Background(), "nonexistent"))
	})
}

func TestMockRateLimiterStore_WithClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewMockRateLimiterStoreWithClock(func() time.Time { return fixedTime })
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 10.0, Burst: 10}

	allowed, err := m.TryTake(context.Background(), "bucket1", 1, config)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestMockRateLimiterStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := NewMockRateLimiterStore()
	ctx := context.Background()
	config := &ratelimiter_dto.TokenBucketConfig{Rate: 1000.0, Burst: 1000}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(index int) {
			defer wg.Done()

			key := fmt.Sprintf("bucket-%d", index%5)
			_, _ = m.TryTake(ctx, key, 1, config)
			_, _ = m.WaitDuration(ctx, key, 1, config)
			_ = m.DeleteBucket(ctx, key)
		}(i)
	}

	wg.Wait()

}

func TestNewMockLLMProvider(t *testing.T) {
	t.Parallel()

	m := NewMockLLMProvider()

	assert.NotNil(t, m)
	assert.True(t, m.SupportsStreamingValue)
	assert.True(t, m.SupportsStructuredValue)
	assert.True(t, m.SupportsToolsValue)
	assert.Nil(t, m.CompleteFunc)
	assert.Nil(t, m.StreamFunc)
	assert.Nil(t, m.ListModelsFunc)
	assert.Nil(t, m.CloseFunc)
	assert.Equal(t, int64(0), m.CompleteCallCount)
	assert.Equal(t, int64(0), m.StreamCallCount)
}

func TestNewMockEmbeddingProvider(t *testing.T) {
	t.Parallel()

	m := NewMockEmbeddingProvider()

	assert.NotNil(t, m)
	assert.Nil(t, m.EmbedFunc)
	assert.Nil(t, m.ListModelsFunc)
	assert.Nil(t, m.CloseFunc)
	assert.Nil(t, m.EmbeddingDimensionsFunc)
	assert.Equal(t, int64(0), m.EmbedCallCount)
}

func TestNewMockCacheStore(t *testing.T) {
	t.Parallel()

	m := NewMockCacheStore()

	assert.NotNil(t, m)
	assert.NotNil(t, m.entries)
	assert.Empty(t, m.entries)
}

func TestNewMockBudgetStore(t *testing.T) {
	t.Parallel()

	m := NewMockBudgetStore()

	assert.NotNil(t, m)
	assert.NotNil(t, m.statuses)
	assert.Empty(t, m.statuses)
}

func TestNewMockMemoryStore(t *testing.T) {
	t.Parallel()

	m := NewMockMemoryStore()

	assert.NotNil(t, m)
	assert.NotNil(t, m.states)
	assert.Empty(t, m.states)
}

func TestNewMockSummariser(t *testing.T) {
	t.Parallel()

	m := NewMockSummariser()

	assert.NotNil(t, m)
	assert.Nil(t, m.CompleteFunc)
	assert.Equal(t, int64(0), m.CompleteCallCount)
}

func TestNewMockRateLimiterStore(t *testing.T) {
	t.Parallel()

	m := NewMockRateLimiterStore()

	assert.NotNil(t, m)
	assert.NotNil(t, m.buckets)
	assert.Empty(t, m.buckets)
	assert.NotNil(t, m.clock)
	assert.Nil(t, m.TryTakeFunc)
	assert.Nil(t, m.WaitDurationFunc)
}

func TestNewMockRateLimiterStoreWithClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixedTime }

	m := NewMockRateLimiterStoreWithClock(clock)

	assert.NotNil(t, m)
	assert.NotNil(t, m.buckets)
	assert.NotNil(t, m.clock)
	assert.Equal(t, fixedTime, m.clock())
}
