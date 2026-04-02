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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_adapters"
	"piko.sh/piko/wdk/clock"
	"piko.sh/piko/wdk/maths"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with defaults", func(t *testing.T) {
		service := NewService("")

		require.NotNil(t, service)
		assert.Empty(t, service.GetDefaultProvider())
	})

	t.Run("creates service with default provider name", func(t *testing.T) {
		service := NewService("openai")

		require.NotNil(t, service)
		assert.Equal(t, "openai", service.GetDefaultProvider())
	})

	t.Run("creates service with custom clock", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		service := NewService("", WithClock(mockClock))

		require.NotNil(t, service)

		assert.NotNil(t, service.GetCostCalculator())
	})

	t.Run("creates service with custom cost calculator", func(t *testing.T) {
		calc := NewCostCalculator()
		service := NewService("", WithCostCalculator(calc))

		require.NotNil(t, service)
		assert.Equal(t, calc, service.GetCostCalculator())
	})

	t.Run("creates service with budget manager", func(t *testing.T) {
		budgetStore := NewMockBudgetStore()
		calc := NewCostCalculator()
		budgetMgr := NewBudgetManager(budgetStore, calc)
		service := NewService("", WithBudgetManager(budgetMgr))

		require.NotNil(t, service)
		assert.Equal(t, budgetMgr, service.GetBudgetManager())
	})

	t.Run("creates service with rate limiter", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		service := NewService("", WithRateLimiter(limiter))

		require.NotNil(t, service)
		assert.Equal(t, limiter, service.GetRateLimiter())
	})
}

func TestNewService_WithAllOptions(t *testing.T) {
	calc := NewCostCalculator()
	budgetStore := NewMockBudgetStore()
	budgetMgr := NewBudgetManager(budgetStore, calc)
	store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
	limiter := NewRateLimiter(store)

	service := NewService("openai", WithCostCalculator(calc), WithBudgetManager(budgetMgr), WithRateLimiter(limiter))

	require.NotNil(t, service)
	assert.Equal(t, "openai", service.GetDefaultProvider())
	assert.Equal(t, calc, service.GetCostCalculator())
	assert.Equal(t, budgetMgr, service.GetBudgetManager())
	assert.Equal(t, limiter, service.GetRateLimiter())
}

func TestService_RegisterProvider(t *testing.T) {
	t.Run("registers provider successfully", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()

		err := service.RegisterProvider(context.Background(), "openai", provider)

		require.NoError(t, err)
		assert.True(t, service.HasProvider("openai"))
	})

	t.Run("returns error for duplicate provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()

		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		err = service.RegisterProvider(context.Background(), "openai", provider)
		assert.ErrorIs(t, err, ErrProviderAlreadyExists)
	})

	t.Run("allows multiple providers", func(t *testing.T) {
		service := NewService("")

		err := service.RegisterProvider(context.Background(), "openai", NewMockLLMProvider())
		require.NoError(t, err)
		err = service.RegisterProvider(context.Background(), "anthropic", NewMockLLMProvider())
		require.NoError(t, err)

		providers := service.GetProviders()
		assert.Len(t, providers, 2)
		assert.Contains(t, providers, "openai")
		assert.Contains(t, providers, "anthropic")
	})
}

func TestService_SetDefaultProvider(t *testing.T) {
	t.Run("sets default provider", func(t *testing.T) {
		service := NewService("")
		err := service.RegisterProvider(context.Background(), "openai", NewMockLLMProvider())
		require.NoError(t, err)

		err = service.SetDefaultProvider(context.Background(), "openai")

		require.NoError(t, err)
		assert.Equal(t, "openai", service.GetDefaultProvider())
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := NewService("")

		err := service.SetDefaultProvider(context.Background(), "non-existent")

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})
}

func TestService_HasProvider(t *testing.T) {
	service := NewService("")
	err := service.RegisterProvider(context.Background(), "openai", NewMockLLMProvider())
	require.NoError(t, err)

	assert.True(t, service.HasProvider("openai"))
	assert.False(t, service.HasProvider("anthropic"))
}

func TestService_Complete(t *testing.T) {
	ctx := context.Background()

	t.Run("completes successfully", func(t *testing.T) {
		service := NewService("openai")
		provider := NewMockLLMProvider()
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		response, err := service.Complete(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "Mock response", response.Content())
	})

	t.Run("returns error when no default provider", func(t *testing.T) {
		service := NewService("")

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		_, err := service.Complete(ctx, request)

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})
}

func TestService_CompleteWithProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("completes with named provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()
		err := service.RegisterProvider(context.Background(), "anthropic", provider)
		require.NoError(t, err)

		request := &llm_dto.CompletionRequest{
			Model:    "claude-3-opus",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		response, err := service.CompleteWithProvider(ctx, "anthropic", request)

		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := NewService("")

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		_, err := service.CompleteWithProvider(ctx, "non-existent", request)

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})

	t.Run("returns error on provider failure", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()
		provider.CompleteFunc = func(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			return nil, errors.New("provider error")
		}
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		_, err = service.CompleteWithProvider(ctx, "openai", request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider error")
	})
}

func TestService_Stream(t *testing.T) {
	ctx := context.Background()

	t.Run("streams successfully", func(t *testing.T) {
		streamCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		service := NewService("openai")
		provider := NewMockLLMProvider()
		provider.SupportsStreamingValue = true
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		events, err := service.Stream(streamCtx, request)

		require.NoError(t, err)
		require.NotNil(t, events)

		for event := range events {
			if event.Done {
				break
			}
		}
	})

	t.Run("returns error when streaming not supported", func(t *testing.T) {
		service := NewService("openai")
		provider := NewMockLLMProvider()
		provider.SupportsStreamingValue = false
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		_, err = service.Stream(ctx, request)

		assert.ErrorIs(t, err, ErrStreamingNotSupported)
	})

	t.Run("returns error when no default provider", func(t *testing.T) {
		service := NewService("")

		request := &llm_dto.CompletionRequest{
			Model:    "gpt-4o",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		_, err := service.Stream(ctx, request)

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})
}

func TestService_Close(t *testing.T) {
	ctx := context.Background()

	t.Run("closes all providers", func(t *testing.T) {
		service := NewService("")
		provider1 := NewMockLLMProvider()
		provider2 := NewMockLLMProvider()

		closeCalled1 := false
		closeCalled2 := false

		provider1.CloseFunc = func(ctx context.Context) error {
			closeCalled1 = true
			return nil
		}
		provider2.CloseFunc = func(ctx context.Context) error {
			closeCalled2 = true
			return nil
		}

		err := service.RegisterProvider(context.Background(), "openai", provider1)
		require.NoError(t, err)
		err = service.RegisterProvider(context.Background(), "anthropic", provider2)
		require.NoError(t, err)

		err = service.Close(ctx)

		require.NoError(t, err)
		assert.True(t, closeCalled1)
		assert.True(t, closeCalled2)
		assert.Empty(t, service.GetProviders())
	})

	t.Run("returns error on close failure", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()
		provider.CloseFunc = func(ctx context.Context) error {
			return errors.New("close error")
		}
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		err = service.Close(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "close error")
	})

	t.Run("joins multiple close errors", func(t *testing.T) {
		service := NewService("")

		p1 := NewMockLLMProvider()
		p1.CloseFunc = func(_ context.Context) error { return errors.New("p1 failed") }

		p2 := NewMockLLMProvider()
		p2.CloseFunc = func(_ context.Context) error { return errors.New("p2 failed") }

		require.NoError(t, service.RegisterProvider(context.Background(), "p1", p1))
		require.NoError(t, service.RegisterProvider(context.Background(), "p2", p2))

		err := service.Close(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "p1 failed")
		assert.Contains(t, err.Error(), "p2 failed")
	})

	t.Run("context cancellation in close", func(t *testing.T) {
		service := NewService("")

		p := NewMockLLMProvider()
		p.CloseFunc = func(_ context.Context) error { return nil }

		require.NoError(t, service.RegisterProvider(context.Background(), "p", p))

		cancelledCtx, cancel := context.WithCancelCause(ctx)
		cancel(fmt.Errorf("test: simulating cancelled context"))

		err := service.Close(cancelledCtx)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestService_SetBudget(t *testing.T) {
	t.Run("sets budget when budget manager exists", func(t *testing.T) {
		budgetStore := NewMockBudgetStore()
		calc := NewCostCalculator()
		budgetMgr := NewBudgetManager(budgetStore, calc)
		service := NewService("", WithBudgetManager(budgetMgr))

		config := &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromMinorInt(10000, "USD"),
		}
		service.SetBudget("test-scope", config)

		assert.True(t, budgetMgr.HasBudget("test-scope"))
	})

	t.Run("does nothing when budget manager is nil", func(t *testing.T) {
		service := NewService("")
		config := &llm_dto.BudgetConfig{
			MaxDailySpend: maths.NewMoneyFromMinorInt(10000, "USD"),
		}

		service.SetBudget("test-scope", config)
	})
}

func TestService_GetBudgetStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("returns status when budget manager exists", func(t *testing.T) {
		budgetStore := NewMockBudgetStore()
		calc := NewCostCalculator()
		budgetMgr := NewBudgetManager(budgetStore, calc)
		service := NewService("", WithBudgetManager(budgetMgr))

		status, err := service.GetBudgetStatus(ctx, "test-scope")

		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "test-scope", status.Scope)
	})

	t.Run("returns empty status when budget manager is nil", func(t *testing.T) {
		service := NewService("")

		status, err := service.GetBudgetStatus(ctx, "test-scope")

		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, "test-scope", status.Scope)
		assert.True(t, status.TotalSpent.CheckIsZero())
	})
}

func TestService_SetPricingTable(t *testing.T) {
	t.Run("sets pricing table", func(t *testing.T) {
		service := NewService("")
		table := &llm_dto.PricingTable{
			Models: []llm_dto.ModelPricing{
				{
					ModelID:         "gpt-4o",
					Provider:        "openai",
					InputCostPer1M:  maths.NewDecimalFromString("2.50"),
					OutputCostPer1M: maths.NewDecimalFromString("10.00"),
				},
			},
		}

		service.SetPricingTable(table)

		calc := service.GetCostCalculator()
		pricing := calc.GetPricingTable()
		assert.NotNil(t, pricing.GetPricing("gpt-4o"))
	})
}

func TestService_SetRateLimits(t *testing.T) {
	t.Run("sets rate limits when rate limiter exists", func(t *testing.T) {
		store := ratelimiter_adapters.NewInMemoryTokenBucketStore()
		limiter := NewRateLimiter(store)
		service := NewService("", WithRateLimiter(limiter))

		service.SetRateLimits("test-scope", 60, 100000)

		assert.True(t, limiter.HasLimits("test-scope"))
	})

	t.Run("does nothing when rate limiter is nil", func(t *testing.T) {
		service := NewService("")

		service.SetRateLimits("test-scope", 60, 100000)
	})
}

func TestService_SetCacheManager(t *testing.T) {
	service := NewService("")
	cacheStore := NewMockCacheStore()
	cacheMgr := NewCacheManager(cacheStore, time.Hour)

	service.SetCacheManager(cacheMgr)

	assert.Equal(t, cacheMgr, service.GetCacheManager())
}

func TestService_NewCompletion(t *testing.T) {
	service := NewService("")

	builder := service.NewCompletion()

	require.NotNil(t, builder)

}

func TestService_NewEmbedding(t *testing.T) {
	service := NewService("")

	builder := service.NewEmbedding()

	require.NotNil(t, builder)
}

func TestService_RegisterEmbeddingProvider(t *testing.T) {
	service := NewService("")
	provider := NewMockEmbeddingProvider()

	err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)

	require.NoError(t, err)
}

func TestService_SetDefaultEmbeddingProvider(t *testing.T) {
	t.Run("sets default embedding provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		err = service.SetDefaultEmbeddingProvider("openai")

		require.NoError(t, err)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := NewService("")

		err := service.SetDefaultEmbeddingProvider("non-existent")

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})
}

func TestService_Embed(t *testing.T) {
	ctx := context.Background()

	t.Run("embeds successfully with default provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)
		err = service.SetDefaultEmbeddingProvider("openai")
		require.NoError(t, err)

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello"},
		}

		response, err := service.Embed(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("returns error when no default provider", func(t *testing.T) {
		service := NewService("")

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello"},
		}

		_, err := service.Embed(ctx, request)

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})
}

func TestService_EmbedWithProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("embeds with named provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello"},
		}

		response, err := service.EmbedWithProvider(ctx, "openai", request)

		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := NewService("")

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello"},
		}

		_, err := service.EmbedWithProvider(ctx, "non-existent", request)

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})
}

func TestService_Complete_UsesProviderDefaultModel(t *testing.T) {
	ctx := context.Background()

	t.Run("resolves empty model from provider default", func(t *testing.T) {
		service := NewService("test")
		provider := NewMockLLMProvider()
		provider.DefaultModelValue = "default-model"

		var capturedModel string
		provider.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			capturedModel = request.Model
			return &llm_dto.CompletionResponse{
				ID:    "mock-response-id",
				Model: request.Model,
				Choices: []llm_dto.Choice{{
					Index:        0,
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Mock response"},
					FinishReason: "stop",
				}},
				Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			}, nil
		}
		require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

		request := &llm_dto.CompletionRequest{
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		response, err := service.Complete(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
		assert.Equal(t, "default-model", capturedModel)
	})

	t.Run("explicit model overrides provider default", func(t *testing.T) {
		service := NewService("test")
		provider := NewMockLLMProvider()
		provider.DefaultModelValue = "default-model"

		var capturedModel string
		provider.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
			capturedModel = request.Model
			return &llm_dto.CompletionResponse{
				ID:    "mock-response-id",
				Model: request.Model,
				Choices: []llm_dto.Choice{{
					Index:        0,
					Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Mock response"},
					FinishReason: "stop",
				}},
				Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
			}, nil
		}
		require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

		request := &llm_dto.CompletionRequest{
			Model:    "explicit-model",
			Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
		}

		response, err := service.Complete(ctx, request)

		require.NoError(t, err)
		require.NotNil(t, response)

		assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
		assert.Equal(t, "explicit-model", capturedModel)
	})
}

func TestService_Stream_UsesProviderDefaultModel(t *testing.T) {
	ctx := context.Background()

	service := NewService("test")
	provider := NewMockLLMProvider()
	provider.DefaultModelValue = "default-model"
	provider.SupportsStreamingValue = true

	var capturedModel string
	provider.StreamFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
		capturedModel = request.Model
		eventChannel := make(chan llm_dto.StreamEvent, 1)
		go func() {
			defer close(eventChannel)
			eventChannel <- llm_dto.NewDoneEvent(nil)
		}()
		return eventChannel, nil
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	request := &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "Hello"}},
	}

	events, err := service.Stream(ctx, request)
	require.NoError(t, err)

	for range events {
	}

	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.StreamCallCount))
	assert.Equal(t, "default-model", capturedModel)
}

func TestService_StreamChannelBuffered(t *testing.T) {
	service := NewService("mock")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "mock", provider))

	provider.SupportsStreamingValue = true

	content := "hello"
	provider.StreamFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
		eventChannel := make(chan llm_dto.StreamEvent, 2)
		go func() {
			defer close(eventChannel)
			eventChannel <- llm_dto.NewChunkEvent(&llm_dto.StreamChunk{
				Delta: &llm_dto.MessageDelta{Content: &content},
			})
			eventChannel <- llm_dto.NewDoneEvent(nil)
		}()
		return eventChannel, nil
	}

	b := service.NewCompletion().
		Model("mock-model").
		User("test")

	eventChannel, err := b.Stream(context.Background())
	require.NoError(t, err)

	var events []llm_dto.StreamEvent
	for event := range eventChannel {
		events = append(events, event)
	}
	require.NotEmpty(t, events)
}

func TestWithPricingTable_SetsCostCalculator(t *testing.T) {
	table := &llm_dto.PricingTable{
		Models: []llm_dto.ModelPricing{
			{
				ModelID:         "gpt-4o",
				Provider:        "openai",
				InputCostPer1M:  maths.NewDecimalFromString("2.50"),
				OutputCostPer1M: maths.NewDecimalFromString("10.00"),
			},
		},
	}

	service := NewService("", WithPricingTable(table))

	assert.NotNil(t, service.GetCostCalculator())
}

func TestWithCircuitBreaker_WrapsProviders(t *testing.T) {
	ctx := context.Background()
	providerErr := errors.New("provider failure")

	service := NewService("test", WithCircuitBreaker(3, 30*time.Second))
	provider := NewMockLLMProvider()
	provider.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, providerErr
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	for range 3 {
		_, err := service.NewCompletion().
			Model("m").
			User("hello").
			Do(ctx)
		require.Error(t, err)
	}

	_, err := service.NewCompletion().
		Model("m").
		User("hello").
		Do(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderOverloaded)
}

func TestService_Complete_NilUsage_NoRateLimiterPanic(t *testing.T) {
	ctx := context.Background()

	service := NewService("test")
	provider := NewMockLLMProvider()
	provider.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{
			ID:    "response-1",
			Model: request.Model,
			Choices: []llm_dto.Choice{{
				Index:        0,
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "ok"},
				FinishReason: "stop",
			}},
			Usage: nil,
		}, nil
	}
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	response, err := service.NewCompletion().
		Model("m").
		User("hello").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)
}
