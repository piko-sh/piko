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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
)

func TestNewEmbeddingService(t *testing.T) {
	t.Run("creates with default clock", func(t *testing.T) {
		service := newEmbeddingService()

		require.NotNil(t, service)
		assert.NotNil(t, service.clock)
		assert.NotNil(t, service.providers)
		assert.Empty(t, service.defaultProvider)
	})

	t.Run("creates with custom clock", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
		service := newEmbeddingService(withEmbeddingServiceClock(mockClock))

		require.NotNil(t, service)
		assert.Equal(t, mockClock, service.clock)
	})
}

func TestEmbeddingService_RegisterEmbeddingProvider(t *testing.T) {
	t.Run("registers provider successfully", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()

		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)

		require.NoError(t, err)
		assert.Len(t, service.providers, 1)
	})

	t.Run("returns error for duplicate provider", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()

		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		err = service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		assert.ErrorIs(t, err, ErrProviderAlreadyExists)
	})

	t.Run("allows multiple different providers", func(t *testing.T) {
		service := newEmbeddingService()

		err := service.RegisterEmbeddingProvider(context.Background(), "openai", NewMockEmbeddingProvider())
		require.NoError(t, err)

		err = service.RegisterEmbeddingProvider(context.Background(), "anthropic", NewMockEmbeddingProvider())
		require.NoError(t, err)

		assert.Len(t, service.providers, 2)
	})
}

func TestEmbeddingService_SetDefaultEmbeddingProvider(t *testing.T) {
	t.Run("sets default provider", func(t *testing.T) {
		service := newEmbeddingService()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", NewMockEmbeddingProvider())
		require.NoError(t, err)

		err = service.SetDefaultEmbeddingProvider("openai")

		require.NoError(t, err)
		assert.Equal(t, "openai", service.defaultProvider)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := newEmbeddingService()

		err := service.SetDefaultEmbeddingProvider("non-existent")

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})
}

func TestEmbeddingService_Embed(t *testing.T) {
	ctx := context.Background()

	t.Run("embeds successfully with named provider", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		response, err := service.Embed(ctx, "openai", request)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Embeddings, 1)
	})

	t.Run("embeds successfully with default provider", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)
		err = service.SetDefaultEmbeddingProvider("openai")
		require.NoError(t, err)

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		response, err := service.Embed(ctx, "", request)

		require.NoError(t, err)
		require.NotNil(t, response)
	})

	t.Run("returns error when no default provider set", func(t *testing.T) {
		service := newEmbeddingService()

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		_, err := service.Embed(ctx, "", request)

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})

	t.Run("returns error for non-existent provider", func(t *testing.T) {
		service := newEmbeddingService()

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		_, err := service.Embed(ctx, "non-existent", request)

		assert.ErrorIs(t, err, ErrProviderNotFound)
	})

	t.Run("returns error when provider fails", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		provider.EmbedFunc = func(ctx context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
			return nil, errors.New("embedding failed")
		}
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		request := &llm_dto.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: []string{"Hello world"},
		}

		_, err = service.Embed(ctx, "openai", request)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embedding failed")
	})
}

func TestEmbeddingService_GetEmbeddingProvider(t *testing.T) {
	t.Run("returns provider by name", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		result, err := service.getEmbeddingProvider("openai")

		require.NoError(t, err)
		assert.Equal(t, provider, result)
	})

	t.Run("returns default provider when name is empty", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)
		err = service.SetDefaultEmbeddingProvider("openai")
		require.NoError(t, err)

		result, err := service.getEmbeddingProvider("")

		require.NoError(t, err)
		assert.Equal(t, provider, result)
	})

	t.Run("returns error when no default and name is empty", func(t *testing.T) {
		service := newEmbeddingService()

		_, err := service.getEmbeddingProvider("")

		assert.ErrorIs(t, err, ErrNoDefaultProvider)
	})
}

func TestNewEmbeddingBuilder(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	require.NotNil(t, builder)
	assert.Equal(t, service, builder.embeddingService)
	assert.NotNil(t, builder.request)
	assert.Empty(t, builder.providerName)
}

func TestEmbeddingBuilder_Model(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.Model("text-embedding-3-small")

	assert.Equal(t, builder, result)
	assert.Equal(t, "text-embedding-3-small", builder.request.Model)
}

func TestEmbeddingBuilder_Input(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.Input("Hello", "World")

	assert.Equal(t, builder, result)
	assert.Equal(t, []string{"Hello", "World"}, builder.request.Input)

	builder.Input("More")
	assert.Equal(t, []string{"Hello", "World", "More"}, builder.request.Input)
}

func TestEmbeddingBuilder_Dimensions(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.Dimensions(1536)

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.request.Dimensions)
	assert.Equal(t, 1536, *builder.request.Dimensions)
}

func TestEmbeddingBuilder_EncodingFormat(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.EncodingFormat("base64")

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.request.EncodingFormat)
	assert.Equal(t, "base64", *builder.request.EncodingFormat)
}

func TestEmbeddingBuilder_Provider(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.Provider("openai")

	assert.Equal(t, builder, result)
	assert.Equal(t, "openai", builder.providerName)
}

func TestEmbeddingBuilder_WithUser(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.User("user-123")

	assert.Equal(t, builder, result)
	require.NotNil(t, builder.request.User)
	assert.Equal(t, "user-123", *builder.request.User)
}

func TestEmbeddingBuilder_WithMetadata(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service)

	result := builder.Metadata("key1", "value1").Metadata("key2", "value2")

	assert.Equal(t, builder, result)
	assert.Equal(t, "value1", builder.request.Metadata["key1"])
	assert.Equal(t, "value2", builder.request.Metadata["key2"])
}

func TestEmbeddingBuilder_Build(t *testing.T) {
	service := newEmbeddingService()
	builder := NewEmbeddingBuilder(service).
		Model("text-embedding-3-small").
		Input("Hello", "World").
		Dimensions(1536)

	request := builder.Build()

	assert.Equal(t, "text-embedding-3-small", request.Model)
	assert.Equal(t, []string{"Hello", "World"}, request.Input)
	require.NotNil(t, request.Dimensions)
	assert.Equal(t, 1536, *request.Dimensions)
}

func TestEmbeddingBuilder_Embed(t *testing.T) {
	ctx := context.Background()

	t.Run("executes embedding request", func(t *testing.T) {
		service := newEmbeddingService()
		provider := NewMockEmbeddingProvider()
		err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
		require.NoError(t, err)

		response, err := NewEmbeddingBuilder(service).
			Model("text-embedding-3-small").
			Input("Hello world").
			Provider("openai").
			Embed(ctx)

		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Len(t, response.Embeddings, 1)
	})

	t.Run("returns error on failure", func(t *testing.T) {
		service := newEmbeddingService()

		_, err := NewEmbeddingBuilder(service).
			Model("text-embedding-3-small").
			Input("Hello world").
			Embed(ctx)

		assert.Error(t, err)
	})
}

func TestEmbeddingBuilder_FullChain(t *testing.T) {
	ctx := context.Background()
	service := newEmbeddingService()
	provider := NewMockEmbeddingProvider()

	var capturedReq *llm_dto.EmbeddingRequest
	provider.EmbedFunc = func(_ context.Context, request *llm_dto.EmbeddingRequest) (*llm_dto.EmbeddingResponse, error) {
		capturedReq = request
		embeddings := make([]llm_dto.Embedding, len(request.Input))
		for i := range request.Input {
			embeddings[i] = llm_dto.Embedding{
				Index:  i,
				Vector: []float32{0.1, 0.2, 0.3},
			}
		}
		return &llm_dto.EmbeddingResponse{
			Model:      request.Model,
			Embeddings: embeddings,
			Usage: &llm_dto.EmbeddingUsage{
				PromptTokens: len(request.Input) * 5,
				TotalTokens:  len(request.Input) * 5,
			},
		}, nil
	}
	err := service.RegisterEmbeddingProvider(context.Background(), "openai", provider)
	require.NoError(t, err)

	response, err := NewEmbeddingBuilder(service).
		Model("text-embedding-3-small").
		Input("Hello").
		Input("World").
		Dimensions(512).
		EncodingFormat("float").
		User("user-123").
		Metadata("env", "test").
		Provider("openai").
		Embed(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)

	assert.Equal(t, int64(1), atomic.LoadInt64(&provider.EmbedCallCount))
	require.NotNil(t, capturedReq)
	assert.Equal(t, "text-embedding-3-small", capturedReq.Model)
	assert.Equal(t, []string{"Hello", "World"}, capturedReq.Input)
	require.NotNil(t, capturedReq.Dimensions)
	assert.Equal(t, 512, *capturedReq.Dimensions)
	require.NotNil(t, capturedReq.EncodingFormat)
	assert.Equal(t, "float", *capturedReq.EncodingFormat)
	require.NotNil(t, capturedReq.User)
	assert.Equal(t, "user-123", *capturedReq.User)
	assert.Equal(t, "test", capturedReq.Metadata["env"])
}
