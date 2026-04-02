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
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestFallback_FirstProviderFails_SecondSucceeds(t *testing.T) {
	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, llm_domain.ErrProviderOverloaded
	}

	service, ctx := createFailingZoltaiService(t, mock)

	response, err := service.NewCompletion().
		User("Test fallback").
		FallbackProviders("failing", "zoltai").
		Do(ctx)

	require.NoError(t, err)
	assert.NotEmpty(t, response.Content())
	assert.NotNil(t, response.FallbackInfo)
	assert.True(t, response.FallbackInfo.WasFallbackUsed())
}

func TestFallback_AllProvidersFail(t *testing.T) {
	mock1 := llm_domain.NewMockLLMProvider()
	mock1.DefaultModelValue = "mock-1"
	mock1.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, llm_domain.ErrProviderOverloaded
	}

	mock2 := llm_domain.NewMockLLMProvider()
	mock2.DefaultModelValue = "mock-2"
	mock2.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, llm_domain.ErrProviderTimeout
	}

	service, ctx := createFailingZoltaiService(t, mock1)
	require.NoError(t, service.RegisterProvider(ctx, "mock2", mock2))

	_, err := service.NewCompletion().
		User("Test all fail").
		FallbackProviders("failing", "mock2").
		Do(ctx)

	require.Error(t, err)
}

func TestFallback_ContextCancellation(t *testing.T) {
	mock := llm_domain.NewMockLLMProvider()
	mock.DefaultModelValue = "mock-1"
	mock.CompleteFunc = func(_ context.Context, _ *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
		return nil, llm_domain.ErrProviderOverloaded
	}

	service, _ := createFailingZoltaiService(t, mock)

	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err := service.NewCompletion().
		User("Test cancellation").
		FallbackProviders("failing", "zoltai").
		Do(ctx)

	assert.ErrorIs(t, err, context.Canceled)
}
