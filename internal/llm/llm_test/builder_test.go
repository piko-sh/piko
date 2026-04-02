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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func TestBuilder_FullFlow_SystemUserMemoryCache(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("Hello there!", 20, 10))

	ctx := context.Background()
	response, err := h.service.NewCompletion().
		Model("test-model").
		System("You are helpful.").
		User("Hi").
		BufferMemory(h.memoryStore, "conv-1", 20).
		Cache(time.Hour).
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "Hello there!", response.Choices[0].Message.Content)

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-model", calls[0].Model)

	require.GreaterOrEqual(t, len(calls[0].Messages), 2)
	assert.Equal(t, llm_dto.RoleSystem, calls[0].Messages[0].Role)
	assert.Equal(t, llm_dto.RoleUser, calls[0].Messages[1].Role)

	state, err := h.memoryStore.Load(ctx, "conv-1")
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.GreaterOrEqual(t, state.MessageCount(), 2)

	assert.GreaterOrEqual(t, h.cacheStore.entryCount(), 1)
}

func TestBuilder_ModelSelection(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	ctx := context.Background()

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)

	calls := h.mockProvider.GetCompleteCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-model", calls[0].Model)
}

func TestBuilder_ProviderSelection(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider2.SetResponse(makeResponse("From fallback", 10, 5))

	ctx := context.Background()
	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		Provider("fallback").
		Do(ctx)

	require.NoError(t, err)
	assert.Equal(t, "From fallback", response.Choices[0].Message.Content)

	assert.Empty(t, h.mockProvider.GetCompleteCalls())

	assert.Len(t, h.mockProvider2.GetCompleteCalls(), 1)
}

func TestBuilder_WithBudgetScope(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("budgeted", 100, 50))

	ctx := context.Background()
	h.service.SetBudget("project-a", &llm_dto.BudgetConfig{})

	response, err := h.service.NewCompletion().
		Model("test-model").
		User("Do something").
		BudgetScope("project-a").
		Do(ctx)

	require.NoError(t, err)
	require.NotNil(t, response)

	status, err := h.service.GetBudgetStatus(ctx, "project-a")
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.Greater(t, status.RequestCount, int64(0))
}

func TestBuilder_CacheAndMemory_Together(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)
	h.mockProvider.SetResponse(makeResponse("cached+memo", 10, 5))
	ctx := context.Background()

	resp1, err := h.service.NewCompletion().
		Model("test-model").
		User("Hello").
		BufferMemory(h.memoryStore, "conv-cm", 20).
		Cache(time.Hour).
		Do(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Equal(t, "cached+memo", resp1.Choices[0].Message.Content)

	assert.Len(t, h.mockProvider.GetCompleteCalls(), 1)

	state, err := h.memoryStore.Load(ctx, "conv-cm")
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.GreaterOrEqual(t, state.MessageCount(), 2)

	assert.GreaterOrEqual(t, h.cacheStore.entryCount(), 1)

}

func TestBuilder_RetryOnTransientError(t *testing.T) {
	skipIfShort(t)

	h := newTestHarness(t)

	h.mockProvider.SetError(llm_domain.ErrProviderOverloaded)

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

	ctx := context.Background()
	_, err := h.service.NewCompletion().
		Model("test-model").
		User("Retry me").
		DefaultRetry().
		Do(ctx)

	require.Error(t, err)

	calls := h.mockProvider.GetCompleteCalls()
	assert.Greater(t, len(calls), 1, "expected multiple calls due to retries")
}
