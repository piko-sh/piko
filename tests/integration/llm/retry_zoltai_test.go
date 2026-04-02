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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestRetry_TransientErrorThenSuccess(t *testing.T) {
	var attempts atomic.Int64

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		n := attempts.Add(1)
		if n < 2 {
			return nil, llm_domain.ErrProviderOverloaded
		}
		return &llm_dto.CompletionResponse{
			Model: "mock-1",
			Choices: []llm_dto.Choice{
				{Message: llm_dto.Message{Role: llm_dto.RoleAssistant, Content: "recovered"}, FinishReason: llm_dto.FinishReasonStop},
			},
			Usage: &llm_dto.Usage{PromptTokens: 10, CompletionTokens: 1, TotalTokens: 11},
		}, nil
	}

	service, ctx := createFailingZoltaiService(t, mock)

	response, err := service.NewCompletion().
		User("Test retry").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    1 * time.Millisecond,
			MaxBackoff:        10 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}).
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "recovered", response.Content())
	assert.GreaterOrEqual(t, attempts.Load(), int64(2))
}

func TestRetry_NonRetryableErrorStops(t *testing.T) {
	var attempts atomic.Int64

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		attempts.Add(1)
		return nil, llm_domain.ErrEmptyModel
	}

	service, ctx := createFailingZoltaiService(t, mock)

	_, err := service.NewCompletion().
		User("Test retry").
		Model("mock-1").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        3,
			InitialBackoff:    1 * time.Millisecond,
			MaxBackoff:        10 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}).
		Do(ctx)

	require.Error(t, err)
	assert.Equal(t, int64(1), attempts.Load())
}

func TestRetry_ExhaustsMaxRetries(t *testing.T) {
	var attempts atomic.Int64

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		attempts.Add(1)
		return nil, llm_domain.ErrProviderOverloaded
	}

	service, ctx := createFailingZoltaiService(t, mock)

	_, err := service.NewCompletion().
		User("Test retry").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        2,
			InitialBackoff:    1 * time.Millisecond,
			MaxBackoff:        10 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}).
		Do(ctx)

	require.Error(t, err)
	assert.Equal(t, int64(3), attempts.Load())
}

func TestRetry_ContextCancellationStopsRetry(t *testing.T) {
	var attempts atomic.Int64

	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		attempts.Add(1)
		return nil, llm_domain.ErrProviderOverloaded
	}

	service, _ := createFailingZoltaiService(t, mock)

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewCompletion().
		User("Test retry").
		Retry(&llm_dto.RetryPolicy{
			MaxRetries:        10,
			InitialBackoff:    1 * time.Millisecond,
			MaxBackoff:        10 * time.Millisecond,
			BackoffMultiplier: 2.0,
		}).
		Do(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}
