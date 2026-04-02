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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestFallback_PrimaryFails_FallbackSucceeds(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetError(errors.New("primary unavailable"))
	h.mockProvider2.SetResponse(makeResponse("fallback response", 10, 5))

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		FallbackProviders("mock", "fallback").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "fallback response", response.Choices[0].Message.Content)

	require.NotNil(t, response.FallbackInfo, "FallbackInfo should be set when fallback was used")
	assert.True(t, response.FallbackInfo.WasFallbackUsed())
	assert.Equal(t, "fallback", response.FallbackInfo.UsedProvider)
	assert.Len(t, response.FallbackInfo.AttemptedProviders, 2)
	assert.Equal(t, "mock", response.FallbackInfo.AttemptedProviders[0])
	assert.Equal(t, "fallback", response.FallbackInfo.AttemptedProviders[1])

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)
	assert.Len(t, h.mockProvider2.GetCompleteCalls(), 1)
}

func TestFallback_AllFail(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetError(errors.New("primary down"))
	h.mockProvider2.SetError(errors.New("fallback down"))

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		FallbackProviders("mock", "fallback").
		Do(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all 2 fallback providers failed")
}

func TestFallback_PrimarySucceeds_NoFallback(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetResponse(makeResponse("primary response", 10, 5))
	h.mockProvider2.SetResponse(makeResponse("fallback response", 10, 5))

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		FallbackProviders("mock", "fallback").
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "primary response", response.Choices[0].Message.Content)

	assert.Nil(t, response.FallbackInfo, "FallbackInfo should be nil when primary succeeds")

	assert.Empty(t, h.mockProvider2.GetCompleteCalls(), "fallback provider should not be called")
	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)
}

func TestFallback_WithRetry(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetError(llm_domain.ErrProviderOverloaded)

	h.mockProvider2.SetResponse(makeResponse("fallback after retries", 10, 5))

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				h.clock.Advance(time.Second)
				time.Sleep(time.Millisecond)
			}
		}
	}()
	defer close(stop)

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		FallbackProviders("mock", "fallback").
		DefaultRetry().
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "fallback after retries", response.Choices[0].Message.Content)

	primaryCalls := h.mockProvider.GetCompleteCalls()
	assert.Greater(t, len(primaryCalls), 1, "primary should be retried before fallback")

	assert.Len(t, h.mockProvider2.GetCompleteCalls(), 1)
}

func TestFallback_RateLimitTrigger(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetError(llm_domain.ErrRateLimited)
	h.mockProvider2.SetResponse(makeResponse("fallback", 10, 5))

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Fallback(llm_dto.NewFallbackConfig("mock", "fallback").
			WithTriggers(llm_dto.FallbackOnRateLimit)).
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "fallback", response.Choices[0].Message.Content)
	require.NotNil(t, response.FallbackInfo)
	assert.True(t, response.FallbackInfo.WasFallbackUsed())
	assert.Equal(t, "fallback", response.FallbackInfo.UsedProvider)
}

func TestFallback_ModelMapping(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	h.mockProvider.SetError(errors.New("primary unavailable"))
	h.mockProvider2.SetResponse(makeResponse("mapped model response", 10, 5))

	config := llm_dto.NewFallbackConfig("mock", "fallback").
		WithModelMapping(map[string]string{
			"fallback": "fallback-specific-model",
		})

	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Fallback(config).
		Do(ctx)

	require.NoError(t, err)

	fallbackCalls := h.mockProvider2.GetCompleteCalls()
	require.Len(t, fallbackCalls, 1)
	assert.Equal(t, "fallback-specific-model", fallbackCalls[0].Model,
		"fallback provider should receive the mapped model name")
}
