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

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	const maxFailures = 3
	inner := NewMockLLMProvider()
	providerErr := errors.New("provider unavailable")
	inner.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, providerErr
	}

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, maxFailures, 30*time.Second)
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "test-model"}

	for range maxFailures {
		_, err := circuitBreaker.Complete(ctx, request)
		require.Error(t, err)
		assert.ErrorIs(t, err, providerErr)
	}

	_, err := circuitBreaker.Complete(ctx, request)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderOverloaded)
}

func TestCircuitBreaker_ExcludesClientErrors(t *testing.T) {
	const maxFailures = 3

	clientErrors := []error{
		context.Canceled,
		context.DeadlineExceeded,
		ErrBudgetExceeded,
		ErrMaxCostExceeded,
		ErrRateLimited,
	}

	for _, clientErr := range clientErrors {
		t.Run(clientErr.Error(), func(t *testing.T) {
			inner := NewMockLLMProvider()
			callCount := 0
			inner.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
				callCount++
				return nil, clientErr
			}

			circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, maxFailures, 30*time.Second)
			ctx := context.Background()
			request := &llm_dto.CompletionRequest{Model: "test-model"}

			for range maxFailures + 2 {
				_, err := circuitBreaker.Complete(ctx, request)
				require.Error(t, err)
				assert.ErrorIs(t, err, clientErr)
			}

			assert.Equal(t, maxFailures+2, callCount)
		})
	}
}

func TestCircuitBreaker_MapsOpenStateToOverloaded(t *testing.T) {
	t.Run("ErrOpenState", func(t *testing.T) {
		mapped := mapCircuitBreakerError(gobreaker.ErrOpenState)
		assert.ErrorIs(t, mapped, ErrProviderOverloaded)
	})

	t.Run("ErrTooManyRequests", func(t *testing.T) {
		mapped := mapCircuitBreakerError(gobreaker.ErrTooManyRequests)
		assert.ErrorIs(t, mapped, ErrProviderOverloaded)
	})

	t.Run("other errors pass through", func(t *testing.T) {
		original := errors.New("some other error")
		mapped := mapCircuitBreakerError(original)
		assert.Equal(t, original, mapped)
	})
}

func TestCircuitBreaker_DelegatesSupportsStreaming(t *testing.T) {
	inner := NewMockLLMProvider()
	inner.SupportsStreamingValue = true

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 5, 30*time.Second)

	assert.True(t, circuitBreaker.SupportsStreaming())

	inner.SupportsStreamingValue = false

	assert.False(t, circuitBreaker.SupportsStreaming())
}

func TestCircuitBreaker_StreamOperation(t *testing.T) {
	const maxFailures = 3
	inner := NewMockLLMProvider()
	providerErr := errors.New("stream provider unavailable")
	inner.StreamFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
		return nil, providerErr
	}

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, maxFailures, 30*time.Second)
	ctx := context.Background()
	request := &llm_dto.CompletionRequest{Model: "test-model"}

	for range maxFailures {
		_, err := circuitBreaker.Stream(ctx, request)
		require.Error(t, err)
		assert.ErrorIs(t, err, providerErr)
	}

	_, err := circuitBreaker.Stream(ctx, request)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProviderOverloaded)

	inner.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return &llm_dto.CompletionResponse{ID: "ok"}, nil
	}

	response, err := circuitBreaker.Complete(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, "ok", response.ID)
}

func TestCircuitBreaker_DelegatesSupportsStructuredOutput(t *testing.T) {
	inner := NewMockLLMProvider()
	inner.SupportsStructuredValue = true

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 3, time.Minute)

	assert.True(t, circuitBreaker.SupportsStructuredOutput())
}

func TestCircuitBreaker_DelegatesSupportsTools(t *testing.T) {
	inner := NewMockLLMProvider()
	inner.SupportsToolsValue = true

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 3, time.Minute)

	assert.True(t, circuitBreaker.SupportsTools())
}

func TestCircuitBreaker_DelegatesListModels(t *testing.T) {
	inner := NewMockLLMProvider()
	expected := []llm_dto.ModelInfo{{Name: "gpt-4"}}
	inner.ListModelsFunc = func(_ context.Context) ([]llm_dto.ModelInfo, error) {
		return expected, nil
	}

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 3, time.Minute)
	ctx := context.Background()

	result, err := circuitBreaker.ListModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestCircuitBreaker_DelegatesClose(t *testing.T) {
	inner := NewMockLLMProvider()
	called := false
	inner.CloseFunc = func(_ context.Context) error {
		called = true
		return nil
	}

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 3, time.Minute)
	ctx := context.Background()

	err := circuitBreaker.Close(ctx)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestCircuitBreaker_DelegatesDefaultModel(t *testing.T) {
	inner := NewMockLLMProvider()
	inner.DefaultModelValue = "gpt-4o"

	circuitBreaker := newCircuitBreakerProvider(context.Background(), "test", inner, 3, time.Minute)

	assert.Equal(t, "gpt-4o", circuitBreaker.DefaultModel())
}
