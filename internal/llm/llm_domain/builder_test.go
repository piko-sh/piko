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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/maths"
)

func TestCompletionBuilder_Model(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Model("gpt-4o")

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Equal(t, "gpt-4o", request.Model)
}

func TestCompletionBuilder_System(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.System("You are a helpful assistant")

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleSystem, request.Messages[0].Role)
	assert.Equal(t, "You are a helpful assistant", request.Messages[0].Content)
}

func TestCompletionBuilder_User(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.User("Hello, world!")

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleUser, request.Messages[0].Role)
	assert.Equal(t, "Hello, world!", request.Messages[0].Content)
}

func TestCompletionBuilder_UserWithImage(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.UserWithImage("Describe this image", "https://example.com/image.png")

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleUser, request.Messages[0].Role)
	require.Len(t, request.Messages[0].ContentParts, 2)
}

func TestCompletionBuilder_UserWithImageData(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.UserWithImageData("Describe this", "image/png", "base64data")

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleUser, request.Messages[0].Role)
}

func TestCompletionBuilder_UserWithImages(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	imagePart := llm_dto.ImageURLPart("https://example.com/image.png")
	result := builder.UserWithImages("Describe these", imagePart)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
}

func TestCompletionBuilder_Assistant(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Assistant("Hello! How can I help?")

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleAssistant, request.Messages[0].Role)
	assert.Equal(t, "Hello! How can I help?", request.Messages[0].Content)
}

func TestCompletionBuilder_Messages(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	messages := []llm_dto.Message{
		llm_dto.NewUserMessage("First"),
		llm_dto.NewAssistantMessage("Response"),
	}

	result := builder.Messages(messages...)

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Len(t, request.Messages, 2)
}

func TestCompletionBuilder_ToolResult(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.ToolResult("call_123", `{"result": "success"}`)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Messages, 1)
	assert.Equal(t, llm_dto.RoleTool, request.Messages[0].Role)
	require.NotNil(t, request.Messages[0].ToolCallID)
	assert.Equal(t, "call_123", *request.Messages[0].ToolCallID)
}

func TestCompletionBuilder_Temperature(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Temperature(0.7)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.Temperature)
	assert.Equal(t, 0.7, *request.Temperature)
}

func TestCompletionBuilder_MaxTokens(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.MaxTokens(1000)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.MaxTokens)
	assert.Equal(t, 1000, *request.MaxTokens)
}

func TestCompletionBuilder_TopP(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.TopP(0.9)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.TopP)
	assert.Equal(t, 0.9, *request.TopP)
}

func TestCompletionBuilder_FrequencyPenalty(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.FrequencyPenalty(0.5)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.FrequencyPenalty)
	assert.Equal(t, 0.5, *request.FrequencyPenalty)
}

func TestCompletionBuilder_PresencePenalty(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.PresencePenalty(0.3)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.PresencePenalty)
	assert.Equal(t, 0.3, *request.PresencePenalty)
}

func TestCompletionBuilder_Stop(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Stop("END", "STOP")

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Equal(t, []string{"END", "STOP"}, request.Stop)
}

func TestCompletionBuilder_Seed(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Seed(42)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.Seed)
	assert.Equal(t, int64(42), *request.Seed)
}

func TestCompletionBuilder_WithTool(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	params := &llm_dto.JSONSchema{
		Type: "object",
		Properties: map[string]*llm_dto.JSONSchema{
			"location": {Type: "string"},
		},
	}

	result := builder.Tool("get_weather", "Get weather for a location", params)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Tools, 1)
	assert.Equal(t, "function", request.Tools[0].Type)
	assert.Equal(t, "get_weather", request.Tools[0].Function.Name)
}

func TestCompletionBuilder_WithStrictTool(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	params := &llm_dto.JSONSchema{
		Type: "object",
		Properties: map[string]*llm_dto.JSONSchema{
			"query": {Type: "string"},
		},
		Required: []string{"query"},
	}

	result := builder.StrictTool("search", "Search for information", params)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.Len(t, request.Tools, 1)
	assert.True(t, *request.Tools[0].Function.Strict)
}

func TestCompletionBuilder_Tools(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	tools := []llm_dto.ToolDefinition{
		llm_dto.NewFunctionTool("tool1", "First tool", nil),
		llm_dto.NewFunctionTool("tool2", "Second tool", nil),
	}

	result := builder.Tools(tools...)

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Len(t, request.Tools, 2)
}

func TestCompletionBuilder_ToolChoice(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	choice := llm_dto.ToolChoiceAuto()

	result := builder.ToolChoice(choice)

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Equal(t, choice, request.ToolChoice)
}

func TestCompletionBuilder_ParallelToolCalls(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.ParallelToolCalls(true)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.ParallelToolCalls)
	assert.True(t, *request.ParallelToolCalls)
}

func TestCompletionBuilder_JSONResponse(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.JSONResponse()

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.ResponseFormat)
	assert.Equal(t, llm_dto.ResponseFormatJSONObject, request.ResponseFormat.Type)
}

func TestCompletionBuilder_StructuredResponse(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	schema := llm_dto.JSONSchema{
		Type: "object",
		Properties: map[string]*llm_dto.JSONSchema{
			"name": {Type: "string"},
		},
	}

	result := builder.StructuredResponse("person", schema)

	assert.Equal(t, builder, result)
	request := builder.Build()
	require.NotNil(t, request.ResponseFormat)
	assert.Equal(t, llm_dto.ResponseFormatJSONSchema, request.ResponseFormat.Type)
}

func TestCompletionBuilder_ResponseFormat(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	format := llm_dto.ResponseFormatJSON()

	result := builder.ResponseFormat(format)

	assert.Equal(t, builder, result)
	request := builder.Build()
	assert.Equal(t, format, request.ResponseFormat)
}

func TestCompletionBuilder_Provider(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Provider("anthropic")

	assert.Equal(t, builder, result)
	assert.Equal(t, "anthropic", builder.providerName)
}

func TestCompletionBuilder_WithProviderOption(t *testing.T) {
	t.Run("initialises map when nil", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		result := builder.ProviderOption("api_version", "2024-01-01")

		assert.Equal(t, builder, result)
		request := builder.Build()
		assert.Equal(t, "2024-01-01", request.ProviderOptions["api_version"])
	})

	t.Run("adds to existing map", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		builder.ProviderOption("key1", "value1")
		builder.ProviderOption("key2", "value2")

		request := builder.Build()
		assert.Equal(t, "value1", request.ProviderOptions["key1"])
		assert.Equal(t, "value2", request.ProviderOptions["key2"])
	})
}

func TestCompletionBuilder_Metadata(t *testing.T) {
	t.Run("initialises map when nil", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		result := builder.Metadata("request_id", "req_123")

		assert.Equal(t, builder, result)
		request := builder.Build()
		assert.Equal(t, "req_123", request.Metadata["request_id"])
	})

	t.Run("adds to existing map", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		builder.Metadata("key1", "value1")
		builder.Metadata("key2", "value2")

		request := builder.Build()
		assert.Equal(t, "value1", request.Metadata["key1"])
		assert.Equal(t, "value2", request.Metadata["key2"])
	})
}

func TestCompletionBuilder_WithBudgetScope(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.BudgetScope("user:123")

	assert.Equal(t, builder, result)
	assert.Equal(t, "user:123", builder.budgetScope)
}

func TestCompletionBuilder_WithMaxCost(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	maxCost := maths.NewMoneyFromMinorInt(1000, "USD")

	result := builder.MaxCost(maxCost)

	assert.Equal(t, builder, result)
	assert.Equal(t, maxCost, builder.maxCost)
}

func TestCompletionBuilder_WithRetry(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	policy := &llm_dto.RetryPolicy{
		MaxRetries:     5,
		InitialBackoff: time.Second,
	}

	result := builder.Retry(policy)

	assert.Equal(t, builder, result)
	assert.Equal(t, policy, builder.retryPolicy)
}

func TestCompletionBuilder_WithDefaultRetry(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.DefaultRetry()

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.retryPolicy)
	assert.Equal(t, 3, builder.retryPolicy.MaxRetries)
}

func TestCompletionBuilder_WithFallback(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	config := &llm_dto.FallbackConfig{
		Providers: []string{"openai", "anthropic"},
	}

	result := builder.Fallback(config)

	assert.Equal(t, builder, result)
	assert.Equal(t, config, builder.fallbackConfig)
}

func TestCompletionBuilder_WithFallbackProviders(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.FallbackProviders("openai", "anthropic", "gemini")

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.fallbackConfig)
	assert.Equal(t, []string{"openai", "anthropic", "gemini"}, builder.fallbackConfig.Providers)
}

func TestCompletionBuilder_Cache(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	result := builder.Cache(time.Hour)

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.cacheConfig)
	assert.True(t, builder.cacheConfig.Enabled)
	assert.Equal(t, time.Hour, builder.cacheConfig.TTL)
}

func TestCompletionBuilder_WithCacheConfig(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	config := &llm_dto.CacheConfig{
		Enabled:   true,
		TTL:       2 * time.Hour,
		Key:       "custom-key",
		SkipRead:  true,
		SkipWrite: false,
	}

	result := builder.CacheConfig(config)

	assert.Equal(t, builder, result)
	assert.Equal(t, config, builder.cacheConfig)
}

func TestCompletionBuilder_WithProviderCache(t *testing.T) {
	t.Run("creates config when nil", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		result := builder.ProviderCache()

		assert.Equal(t, builder, result)
		require.NotNil(t, builder.cacheConfig)
		assert.True(t, builder.cacheConfig.UseProviderCache)
	})

	t.Run("updates existing config", func(t *testing.T) {
		service := NewService("")
		builder := service.NewCompletion()

		builder.Cache(time.Hour)
		builder.ProviderCache()

		assert.True(t, builder.cacheConfig.Enabled)
		assert.True(t, builder.cacheConfig.UseProviderCache)
	})
}

func TestCompletionBuilder_Memory(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	store := NewMockMemoryStore()
	memory := NewBufferMemory(store, WithBufferSize(10))

	result := builder.Memory(memory, "conv_123")

	assert.Equal(t, builder, result)
	assert.Equal(t, memory, builder.memory)
	assert.Equal(t, "conv_123", builder.conversationID)
}

func TestCompletionBuilder_WithBufferMemory(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion()

	store := NewMockMemoryStore()

	result := builder.BufferMemory(store, "conv_456", 20)

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.memory)
	assert.Equal(t, "conv_456", builder.conversationID)
}

func TestCompletionBuilder_Build(t *testing.T) {
	service := NewService("")
	builder := service.NewCompletion().
		Model("gpt-4o").
		System("You are helpful").
		User("Hello").
		Temperature(0.7).
		MaxTokens(500)

	request := builder.Build()

	assert.Equal(t, "gpt-4o", request.Model)
	assert.Len(t, request.Messages, 2)
	require.NotNil(t, request.Temperature)
	assert.Equal(t, 0.7, *request.Temperature)
	require.NotNil(t, request.MaxTokens)
	assert.Equal(t, 500, *request.MaxTokens)
}

func TestCompletionBuilder_Complete(t *testing.T) {
	ctx := context.Background()

	t.Run("executes successfully", func(t *testing.T) {
		service := NewService("openai")
		provider := NewMockLLMProvider()
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		response, err := service.NewCompletion().
			Model("gpt-4o").
			User("Hello").
			Do(ctx)

		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("uses specified provider", func(t *testing.T) {
		service := NewService("")
		provider := NewMockLLMProvider()
		err := service.RegisterProvider(context.Background(), "anthropic", provider)
		require.NoError(t, err)

		response, err := service.NewCompletion().
			Model("claude-3-opus").
			User("Hello").
			Provider("anthropic").
			Do(ctx)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
	})

	t.Run("returns error when no provider", func(t *testing.T) {
		service := NewService("")

		_, err := service.NewCompletion().
			Model("gpt-4o").
			User("Hello").
			Do(ctx)

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})
}

func TestCompletionBuilder_Stream(t *testing.T) {
	ctx := context.Background()

	t.Run("streams successfully", func(t *testing.T) {
		streamCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		service := NewService("openai")
		provider := NewMockLLMProvider()
		provider.SupportsStreamingValue = true
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		events, err := service.NewCompletion().
			Model("gpt-4o").
			User("Hello").
			Stream(streamCtx)

		require.NoError(t, err)
		require.NotNil(t, events)

		for event := range events {
			if event.Done {
				break
			}
		}
	})

	t.Run("uses specified provider", func(t *testing.T) {
		streamCtx, cancel := context.WithCancelCause(ctx)
		defer cancel(fmt.Errorf("test: cleanup"))

		service := NewService("")
		provider := NewMockLLMProvider()
		provider.SupportsStreamingValue = true
		err := service.RegisterProvider(context.Background(), "anthropic", provider)
		require.NoError(t, err)

		events, err := service.NewCompletion().
			Model("claude-3-opus").
			User("Hello").
			Provider("anthropic").
			Stream(streamCtx)

		require.NoError(t, err)
		require.NotNil(t, events)
	})

	t.Run("returns error when streaming not supported", func(t *testing.T) {
		service := NewService("openai")
		provider := NewMockLLMProvider()
		provider.SupportsStreamingValue = false
		err := service.RegisterProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		_, err = service.NewCompletion().
			Model("gpt-4o").
			User("Hello").
			Stream(ctx)

		assert.ErrorIs(t, err, ErrStreamingNotSupported)
	})
}

func TestCompletionBuilder_FullChain(t *testing.T) {
	ctx := context.Background()

	service := NewService("openai")
	provider := NewMockLLMProvider()

	var capturedReq *llm_dto.CompletionRequest
	provider.CompleteFunc = func(_ context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		capturedReq = request
		return &llm_dto.CompletionResponse{
			ID:      "mock-response-id",
			Model:   request.Model,
			Created: time.Now().Unix(),
			Choices: []llm_dto.Choice{{
				Index:        0,
				Message:      llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "Mock response"},
				FinishReason: "stop",
			}},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
		}, nil
	}
	err := service.RegisterProvider(context.Background(), "openai", provider)
	require.NoError(t, err)

	response, err := service.NewCompletion().
		Model("gpt-4o").
		System("You are a helpful assistant").
		User("What is 2+2?").
		Temperature(0.5).
		MaxTokens(100).
		TopP(0.9).
		Metadata("request_id", "test-123").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)

	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
	require.NotNil(t, capturedReq)
	assert.Equal(t, "gpt-4o", capturedReq.Model)
	assert.Len(t, capturedReq.Messages, 2)
	require.NotNil(t, capturedReq.Temperature)
	assert.Equal(t, 0.5, *capturedReq.Temperature)
}

func TestCompletionBuilder_WithCacheAndExecution(t *testing.T) {
	ctx := context.Background()

	service := NewService("openai")
	provider := NewMockLLMProvider()
	err := service.RegisterProvider(context.Background(), "openai", provider)
	require.NoError(t, err)

	cacheStore := NewMockCacheStore()
	cacheMgr := NewCacheManager(cacheStore, time.Hour)
	service.SetCacheManager(cacheMgr)

	resp1, err := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Cache(time.Hour).
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))

	resp2, err := service.NewCompletion().
		Model("gpt-4o").
		User("Hello").
		Cache(time.Hour).
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, resp2)

	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.CompleteCallCount))
}

func TestCompletionBuilder_WithMemoryAndExecution(t *testing.T) {
	ctx := context.Background()

	service := NewService("openai")
	provider := NewMockLLMProvider()
	err := service.RegisterProvider(context.Background(), "openai", provider)
	require.NoError(t, err)

	store := NewMockMemoryStore()
	memory := NewBufferMemory(store, WithBufferSize(10))

	_, err = service.NewCompletion().
		Model("gpt-4o").
		User("Hello, my name is Alice").
		Memory(memory, "conv_123").
		Do(ctx)

	require.NoError(t, err)

	messages, err := memory.GetMessages(ctx, "conv_123")
	require.NoError(t, err)
	assert.NotEmpty(t, messages)
}

func TestCompletionBuilder_DryRun(t *testing.T) {
	ctx := context.Background()

	service := NewService("")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	dump := service.NewCompletion().
		Model("m").
		Provider("test").
		System("system prompt").
		User("user prompt").
		DryRun(ctx)

	require.NotNil(t, dump)
	assert.Equal(t, "m", dump.Model)
	assert.Equal(t, "test", dump.Provider)
	require.Len(t, dump.Messages, 2)
	assert.Equal(t, llm_dto.RoleSystem, dump.Messages[0].Role)
	assert.Equal(t, "user prompt", dump.Messages[1].Content)
}

func TestCompletionBuilder_DryRun_IncludesTools(t *testing.T) {
	ctx := context.Background()

	service := NewService("")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	dump := service.NewCompletion().
		Model("m").
		Provider("test").
		User("hello").
		Tools(llm_dto.NewFunctionTool("fn", "desc", nil)).
		DryRun(ctx)

	require.NotNil(t, dump)
	require.Len(t, dump.Tools, 1)
	assert.Equal(t, "fn", dump.Tools[0].Function.Name)
}

func TestCompletionBuilder_DryRunRedacted_TruncatesLongContent(t *testing.T) {
	ctx := context.Background()

	service := NewService("")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	longContent := strings.Repeat("a", 100)

	dump := service.NewCompletion().
		Model("m").
		Provider("test").
		System(longContent).
		DryRunRedacted(ctx)

	require.NotNil(t, dump)
	require.Len(t, dump.Messages, 1)
	assert.Len(t, dump.Messages[0].Content, 53)
	assert.True(t, strings.HasSuffix(dump.Messages[0].Content, "..."))
}

func TestCompletionBuilder_DryRunRedacted_ShortContentUnchanged(t *testing.T) {
	ctx := context.Background()

	service := NewService("")
	provider := NewMockLLMProvider()
	require.NoError(t, service.RegisterProvider(context.Background(), "test", provider))

	dump := service.NewCompletion().
		Model("m").
		Provider("test").
		User("short").
		DryRunRedacted(ctx)

	require.NotNil(t, dump)
	require.Len(t, dump.Messages, 1)
	assert.Equal(t, "short", dump.Messages[0].Content)
}
